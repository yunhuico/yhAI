-- Copyright (C) Yichun Zhang (agentzh)


local ffi = require 'ffi'
local base = require "resty.core.base"
local bit = require "bit"
require "resty.core.time"  -- for ngx.now used by resty.lrucache
local lrucache = require "resty.lrucache"

local lrucache_get = lrucache.get
local lrucache_set = lrucache.set
local ffi_string = ffi.string
local ffi_new = ffi.new
local ffi_gc = ffi.gc
local ffi_copy = ffi.copy
local ffi_cast = ffi.cast
local C = ffi.C
local bor = bit.bor
local band = bit.band
local lshift = bit.lshift
local sub = string.sub
local fmt = string.format
local byte = string.byte
local setmetatable = setmetatable
local concat = table.concat
local ngx = ngx
local type = type
local tostring = tostring
local error = error
local get_string_buf = base.get_string_buf
local get_string_buf_size = base.get_string_buf_size
local get_size_ptr = base.get_size_ptr
local new_tab = base.new_tab
local floor = math.floor
local print = print
local tonumber = tonumber
local ngx_log = ngx.log
local ngx_ERR = ngx.ERR


if not ngx.re then
    ngx.re = {}
end


local MAX_ERR_MSG_LEN = 128


local FLAG_COMPILE_ONCE  = 0x01
local FLAG_DFA           = 0x02
local FLAG_JIT           = 0x04
local FLAG_DUPNAMES      = 0x08
local FLAG_NO_UTF8_CHECK = 0x10


local PCRE_CASELESS          = 0x0000001
local PCRE_MULTILINE         = 0x0000002
local PCRE_DOTALL            = 0x0000004
local PCRE_EXTENDED          = 0x0000008
local PCRE_ANCHORED          = 0x0000010
local PCRE_UTF8              = 0x0000800
local PCRE_DUPNAMES          = 0x0080000
local PCRE_JAVASCRIPT_COMPAT = 0x2000000


local PCRE_ERROR_NOMATCH = -1


local regex_match_cache
local regex_sub_func_cache = new_tab(0, 4)
local regex_sub_str_cache = new_tab(0, 4)
local max_regex_cache_size
local regex_cache_size = 0
local script_engine


ffi.cdef[[
    typedef struct {
        ngx_str_t                   value;
        void                       *lengths;
        void                       *values;
    } ngx_http_lua_complex_value_t;

    typedef struct {
        void                         *pool;
        unsigned char                *name_table;
        int                           name_count;
        int                           name_entry_size;

        int                           ncaptures;
        int                          *captures;

        void                         *regex;
        void                         *regex_sd;

        ngx_http_lua_complex_value_t *replace;

        const char                   *pattern;
    } ngx_http_lua_regex_t;

    ngx_http_lua_regex_t *
        ngx_http_lua_ffi_compile_regex(const unsigned char *pat,
            size_t pat_len, int flags,
            int pcre_opts, unsigned char *errstr,
            size_t errstr_size);

    int ngx_http_lua_ffi_exec_regex(ngx_http_lua_regex_t *re, int flags,
        const unsigned char *s, size_t len, int pos);

    void ngx_http_lua_ffi_destroy_regex(ngx_http_lua_regex_t *re);

    int ngx_http_lua_ffi_compile_replace_template(ngx_http_lua_regex_t *re,
                                                  const unsigned char
                                                  *replace_data,
                                                  size_t replace_len);

    struct ngx_http_lua_script_engine_s;
    typedef struct ngx_http_lua_script_engine_s  *ngx_http_lua_script_engine_t;

    ngx_http_lua_script_engine_t *ngx_http_lua_ffi_create_script_engine(void);

    void ngx_http_lua_ffi_init_script_engine(ngx_http_lua_script_engine_t *e,
                                             const unsigned char *subj,
                                             ngx_http_lua_regex_t *compiled,
                                             int count);

    void ngx_http_lua_ffi_destroy_script_engine(
        ngx_http_lua_script_engine_t *e);

    size_t ngx_http_lua_ffi_script_eval_len(ngx_http_lua_script_engine_t *e,
                                            ngx_http_lua_complex_value_t *cv);

    size_t ngx_http_lua_ffi_script_eval_data(ngx_http_lua_script_engine_t *e,
                                             ngx_http_lua_complex_value_t *cv,
                                             unsigned char *dst);

    uint32_t ngx_http_lua_ffi_max_regex_cache_size(void);
]]


local c_str_type = ffi.typeof("const char *")

local cached_re_opts = new_tab(0, 4)

local _M = {
    version = base.version
}


local buf_grow_ratio = 2

function _M.set_buf_grow_ratio(ratio)
    buf_grow_ratio = ratio
end


local function get_max_regex_cache_size()
    if max_regex_cache_size then
        return max_regex_cache_size
    end
    max_regex_cache_size = C.ngx_http_lua_ffi_max_regex_cache_size()
    return max_regex_cache_size
end


local function parse_regex_opts(opts)
    local t = cached_re_opts[opts]
    if t then
        return t[1], t[2]
    end

    local flags = 0
    local pcre_opts = 0
    local len = #opts

    for i = 1, len do
        local opt = byte(opts, i)
        if opt == byte("o") then
            flags = bor(flags, FLAG_COMPILE_ONCE)

        elseif opt == byte("j") then
            flags = bor(flags, FLAG_JIT)

        elseif opt == byte("i") then
            pcre_opts = bor(pcre_opts, PCRE_CASELESS)

        elseif opt == byte("s") then
            pcre_opts = bor(pcre_opts, PCRE_DOTALL)

        elseif opt == byte("m") then
            pcre_opts = bor(pcre_opts, PCRE_MULTILINE)

        elseif opt == byte("u") then
            pcre_opts = bor(pcre_opts, PCRE_UTF8)

        elseif opt == byte("U") then
            pcre_opts = bor(pcre_opts, PCRE_UTF8)
            flags = bor(flags, FLAG_NO_UTF8_CHECK)

        elseif opt == byte("x") then
            pcre_opts = bor(pcre_opts, PCRE_EXTENDED)

        elseif opt == byte("d") then
            flags = bor(flags, FLAG_DFA)

        elseif opt == byte("a") then
            pcre_opts = bor(pcre_opts, PCRE_ANCHORED)

        elseif opt == byte("D") then
            pcre_opts = bor(pcre_opts, PCRE_DUPNAMES)
            flags = bor(flags, FLAG_DUPNAMES)

        elseif opt == byte("J") then
            pcre_opts = bor(pcre_opts, PCRE_JAVASCRIPT_COMPAT)

        else
            return error(fmt('unknown flag "%s" (flags "%s")',
                             sub(opts, i, i), opts))
        end
    end

    cached_re_opts[opts] = {flags, pcre_opts}
    return flags, pcre_opts
end


local function collect_named_captures(compiled, flags, res)
    local name_count = compiled.name_count
    local name_table = compiled.name_table
    local entry_size = compiled.name_entry_size

    local ind = 0
    local dup_names = (band(flags, FLAG_DUPNAMES) ~= 0)
    for i = 1, name_count do
        local n = bor(lshift(name_table[ind], 8), name_table[ind + 1])
        -- ngx.say("n = ", n)
        local name = ffi_string(name_table + ind + 2)
        local cap = res[n]
        if cap then
            if dup_names then
                local old = res[name]
                if old then
                    old[#old + 1] = cap
                else
                    res[name] = {cap}
                end
            else
                res[name] = cap
            end
        end

        ind = ind + entry_size
    end
end


local function collect_captures(compiled, rc, subj, flags, res)
    local cap = compiled.captures
    local name_count = compiled.name_count

    if not res then
        res = new_tab(rc, name_count)
    end

    local i = 0
    local n = 0
    while i < rc do
        local from = cap[n]
        if from >= 0 then
            local to = cap[n + 1]
            res[i] = sub(subj, from + 1, to)
        end
        i = i + 1
        n = n + 2
    end

    if name_count > 0 then
        collect_named_captures(compiled, flags, res)
    end

    return res
end


local function destroy_compiled_regex(compiled)
    C.ngx_http_lua_ffi_destroy_regex(ffi_gc(compiled, nil))
end


local function re_match_compile(regex, opts)
    local flags = 0
    local pcre_opts = 0

    if opts then
        flags, pcre_opts = parse_regex_opts(opts)
    else
        opts = ""
    end

    local compiled, key
    local compile_once = (band(flags, FLAG_COMPILE_ONCE) == 1)

    -- FIXME: better put this in the outer scope when fixing the ngx.re API's
    -- compatibility in the init_by_lua* context.
    if not regex_match_cache then
        local sz = get_max_regex_cache_size()
        if sz <= 0 then
            compile_once = false
        else
            regex_match_cache = lrucache.new(sz)
        end
    end

    if compile_once then
        key = regex .. '\0' .. opts
        compiled = lrucache_get(regex_match_cache, key)
    end

    -- compile the regex

    if compiled == nil then
        -- print("compiled regex not found, compiling regex...")
        local errbuf = get_string_buf(MAX_ERR_MSG_LEN)

        compiled = C.ngx_http_lua_ffi_compile_regex(regex, #regex,
                                                    flags, pcre_opts,
                                                    errbuf, MAX_ERR_MSG_LEN)

        if compiled == nil then
            return nil, ffi_string(errbuf)
        end

        ffi_gc(compiled, C.ngx_http_lua_ffi_destroy_regex)

        -- print("ncaptures: ", compiled.ncaptures)

        if compile_once then
            -- print("inserting compiled regex into cache")
            lrucache_set(regex_match_cache, key, compiled)
        end
    end

    return compiled, compile_once, flags
end


local function re_match_helper(subj, regex, opts, ctx, want_caps, res, nth)
    local compiled, compile_once, flags = re_match_compile(regex, opts)
    if compiled == nil then
        -- compiled_once holds the error string
        if not want_caps then
            return nil, nil, compile_once
        end
        return nil, compile_once
    end

    -- exec the compiled regex

    local rc
    do
        local pos
        if ctx then
            pos = ctx.pos
            if not pos or pos <= 0 then
                pos = 0
            else
                pos = pos - 1
            end

        else
            pos = 0
        end

        rc = C.ngx_http_lua_ffi_exec_regex(compiled, flags, subj, #subj, pos)
    end

    if rc == PCRE_ERROR_NOMATCH then
        if not compile_once then
            destroy_compiled_regex(compiled)
        end
        return nil
    end

    if rc < 0 then
        if not compile_once then
            destroy_compiled_regex(compiled)
        end
        if not want_caps then
            return nil, nil, "pcre_exec() failed: " .. rc
        end
        return nil, "pcre_exec() failed: " .. rc
    end

    if rc == 0 then
        if band(flags, FLAG_DFA) == 0 then
            if not want_caps then
                return nil, nil, "capture size too small"
            end
            return nil, "capture size too small"
        end

        rc = 1
    end

    -- print("cap 0: ", compiled.captures[0])
    -- print("cap 1: ", compiled.captures[1])

    if ctx then
        ctx.pos = compiled.captures[1] + 1
    end

    if not want_caps then
        if not nth or nth < 0 then
            nth = 0
        end

        if nth > compiled.ncaptures then
            return nil, nil, "nth out of bound"
        end

        if nth >= rc then
            return nil, nil
        end

        local from = compiled.captures[nth * 2] + 1
        local to = compiled.captures[nth * 2 + 1]

        if from < 0 or to < 0 then
            return nil, nil
        end

        return from, to
    end

    res = collect_captures(compiled, rc, subj, flags, res)

    if not compile_once then
        destroy_compiled_regex(compiled)
    end

    return res
end


function ngx.re.match(subj, regex, opts, ctx, res)
    return re_match_helper(subj, regex, opts, ctx, true, res)
end


function ngx.re.find(subj, regex, opts, ctx, nth)
    return re_match_helper(subj, regex, opts, ctx, false, nil, nth)
end


local function new_script_engine(subj, compiled, count)
    if not script_engine then
        script_engine = C.ngx_http_lua_ffi_create_script_engine()
        if script_engine == nil then
            return nil
        end
        ffi_gc(script_engine, C.ngx_http_lua_ffi_destroy_script_engine)
    end

    C.ngx_http_lua_ffi_init_script_engine(script_engine, subj, compiled,
                                          count)
    return script_engine
end


local function check_buf_size(buf, buf_size, pos, len, new_len, must_alloc)
    if new_len > buf_size then
        buf_size = buf_size * buf_grow_ratio
        if buf_size < new_len then
            buf_size = new_len
        end
        local new_buf = get_string_buf(buf_size, must_alloc)
        ffi_copy(new_buf, buf, len)
        buf = new_buf
        pos = buf + len
    end
    return buf, buf_size, pos, new_len
end


local function re_sub_compile(regex, opts, replace, func)
    local flags = 0
    local pcre_opts = 0

    if opts then
        flags, pcre_opts = parse_regex_opts(opts)
    else
        opts = ""
    end

    local compiled
    local compile_once = (band(flags, FLAG_COMPILE_ONCE) == 1)
    if compile_once then
        if func then
            local subcache = regex_sub_func_cache[opts]
            if subcache then
                -- print("cache hit!")
                compiled = subcache[regex]
            end

        else
            local subcache = regex_sub_str_cache[opts]
            if subcache then
                local subsubcache = subcache[regex]
                if subsubcache then
                    -- print("cache hit!")
                    compiled = subsubcache[replace]
                end
            end
        end
    end

    -- compile the regex

    if compiled == nil then
        -- print("compiled regex not found, compiling regex...")
        local errbuf = get_string_buf(MAX_ERR_MSG_LEN)

        compiled = C.ngx_http_lua_ffi_compile_regex(regex, #regex, flags,
                                                    pcre_opts, errbuf,
                                                    MAX_ERR_MSG_LEN)

        if compiled == nil then
            return nil, ffi_string(errbuf)
        end

        ffi_gc(compiled, C.ngx_http_lua_ffi_destroy_regex)

        if func == nil then
            local rc =
                C.ngx_http_lua_ffi_compile_replace_template(compiled,
                                                            replace, #replace)
            if rc ~= 0 then
                if not compile_once then
                    destroy_compiled_regex(compiled)
                end
                return nil, "failed to compile the replacement template"
            end
        end

        -- print("ncaptures: ", compiled.ncaptures)

        if compile_once then
            if regex_cache_size < get_max_regex_cache_size() then
                -- print("inserting compiled regex into cache")
                if func then
                    local subcache = regex_sub_func_cache[opts]
                    if not subcache then
                        regex_sub_func_cache[opts] = {[regex] = compiled}

                    else
                        subcache[regex] = compiled
                    end

                else
                    local subcache = regex_sub_str_cache[opts]
                    if not subcache then
                        regex_sub_str_cache[opts] =
                            {[regex] = {[replace] = compiled}}

                    else
                        local subsubcache = subcache[regex]
                        if not subsubcache then
                            subcache[regex] = {[replace] = compiled}

                        else
                            subsubcache[replace] = compiled
                        end
                    end
                end

                regex_cache_size = regex_cache_size + 1
            else
                compile_once = false
            end
        end
    end

    return compiled, compile_once, flags
end


local function re_sub_func_helper(subj, regex, replace, opts, global)
    local compiled, compile_once, flags =
                                    re_sub_compile(regex, opts, nil, replace)
    if not compiled then
        -- error string is in compile_once
        return nil, nil, compile_once
    end

    -- exec the compiled regex

    local subj_len = #subj
    local count = 0
    local pos = 0
    local cp_pos = 0

    local dst_buf_size = get_string_buf_size()
    -- Note: we have to always allocate the string buffer because
    -- the user might call whatever resty.core's API functions recursively
    -- in the user callback function.
    local dst_buf = get_string_buf(dst_buf_size, true)
    local dst_pos = dst_buf
    local dst_len = 0

    while true do
        local rc = C.ngx_http_lua_ffi_exec_regex(compiled, flags, subj,
                                                 subj_len, pos)
        if rc == PCRE_ERROR_NOMATCH then
            break
        end

        if rc < 0 then
            if not compile_once then
                destroy_compiled_regex(compiled)
            end
            return nil, nil, "pcre_exec() failed: " .. rc
        end

        if rc == 0 then
            if band(flags, FLAG_DFA) == 0 then
                if not compile_once then
                    destroy_compiled_regex(compiled)
                end
                return nil, nil, "capture size too small"
            end

            rc = 1
        end

        count = count + 1
        local prefix_len = compiled.captures[0] - cp_pos

        local res = collect_captures(compiled, rc, subj, flags)

        local bit = replace(res)
        local bit_len = #bit

        local new_dst_len = dst_len + prefix_len + bit_len
        dst_buf, dst_buf_size, dst_pos, dst_len =
            check_buf_size(dst_buf, dst_buf_size, dst_pos, dst_len,
                           new_dst_len, true)

        if prefix_len > 0 then
            ffi_copy(dst_pos, ffi_cast(c_str_type, subj) + cp_pos,
                     prefix_len)
            dst_pos = dst_pos + prefix_len
        end

        if bit_len > 0 then
            ffi_copy(dst_pos, bit, bit_len)
            dst_pos = dst_pos + bit_len
        end

        cp_pos = compiled.captures[1]
        pos = cp_pos
        if pos == compiled.captures[0] then
            pos = pos + 1
            if pos > subj_len then
                break
            end
        end

        if not global then
            break
        end
    end

    if not compile_once then
        destroy_compiled_regex(compiled)
    end

    if count > 0 then
        if pos < subj_len then
            local suffix_len = subj_len - cp_pos

            local new_dst_len = dst_len + suffix_len
            dst_buf, dst_buf_size, dst_pos, dst_len =
                check_buf_size(dst_buf, dst_buf_size, dst_pos, dst_len,
                               new_dst_len, true)

            ffi_copy(dst_pos, ffi_cast(c_str_type, subj) + cp_pos,
                     suffix_len)
        end
        return ffi_string(dst_buf, dst_len), count
    end

    return subj, 0
end


local function re_sub_str_helper(subj, regex, replace, opts, global)
    local compiled, compile_once, flags =
                                    re_sub_compile(regex, opts, replace, nil)
    if not compiled then
        -- error string is in compile_once
        return nil, nil, compile_once
    end

    -- exec the compiled regex

    local subj_len = #subj
    local count = 0
    local pos = 0
    local cp_pos = 0

    local dst_buf_size = get_string_buf_size()
    local dst_buf = get_string_buf(dst_buf_size)
    local dst_pos = dst_buf
    local dst_len = 0

    while true do
        local rc = C.ngx_http_lua_ffi_exec_regex(compiled, flags, subj,
                                                 subj_len, pos)
        if rc == PCRE_ERROR_NOMATCH then
            break
        end

        if rc < 0 then
            if not compile_once then
                destroy_compiled_regex(compiled)
            end
            return nil, nil, "pcre_exec() failed: " .. rc
        end

        if rc == 0 then
            if band(flags, FLAG_DFA) == 0 then
                if not compile_once then
                    destroy_compiled_regex(compiled)
                end
                return nil, nil, "capture size too small"
            end

            rc = 1
        end

        count = count + 1
        local prefix_len = compiled.captures[0] - cp_pos

        local cv = compiled.replace
        if cv.lengths ~= nil then
            local e = new_script_engine(subj, compiled, rc)
            if e == nil then
                return nil, nil, "failed to create script engine"
            end

            local bit_len = C.ngx_http_lua_ffi_script_eval_len(e, cv)
            local new_dst_len = dst_len + prefix_len + bit_len
            dst_buf, dst_buf_size, dst_pos, dst_len =
                check_buf_size(dst_buf, dst_buf_size, dst_pos, dst_len,
                               new_dst_len)

            if prefix_len > 0 then
                ffi_copy(dst_pos, ffi_cast(c_str_type, subj) + cp_pos,
                         prefix_len)
                dst_pos = dst_pos + prefix_len
            end

            if bit_len > 0 then
                C.ngx_http_lua_ffi_script_eval_data(e, cv, dst_pos)
                dst_pos = dst_pos + bit_len
            end

        else
            local bit_len = cv.value.len

            dst_buf, dst_buf_size, dst_pos, dst_len =
                check_buf_size(dst_buf, dst_buf_size, dst_pos, dst_len,
                               dst_len + prefix_len + bit_len)

            if prefix_len > 0 then
                ffi_copy(dst_pos, ffi_cast(c_str_type, subj) + cp_pos,
                         prefix_len)
                dst_pos = dst_pos + prefix_len
            end

            if bit_len > 0 then
                ffi_copy(dst_pos, cv.value.data, bit_len)
                dst_pos = dst_pos + bit_len
            end
        end

        cp_pos = compiled.captures[1]
        pos = cp_pos
        if pos == compiled.captures[0] then
            pos = pos + 1
            if pos > subj_len then
                break
            end
        end

        if not global then
            break
        end
    end

    if not compile_once then
        destroy_compiled_regex(compiled)
    end

    if count > 0 then
        if pos < subj_len then
            local suffix_len = subj_len - cp_pos

            local new_dst_len = dst_len + suffix_len
            dst_buf, dst_buf_size, dst_pos, dst_len =
                check_buf_size(dst_buf, dst_buf_size, dst_pos, dst_len,
                               new_dst_len)

            ffi_copy(dst_pos, ffi_cast(c_str_type, subj) + cp_pos,
                     suffix_len)
        end
        return ffi_string(dst_buf, dst_len), count
    end

    return subj, 0
end


local function re_sub_helper(subj, regex, replace, opts, global)
    local repl_type = type(replace)
    if repl_type == "function" then
        return re_sub_func_helper(subj, regex, replace, opts, global)
    end

    if repl_type ~= "string" then
        replace = tostring(replace)
    end

    return re_sub_str_helper(subj, regex, replace, opts, global)
end


function ngx.re.sub(subj, regex, replace, opts)
    return re_sub_helper(subj, regex, replace, opts, false)
end


function ngx.re.gsub(subj, regex, replace, opts)
    return re_sub_helper(subj, regex, replace, opts, true)
end


return _M
