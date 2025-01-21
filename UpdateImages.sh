#!/bin/bash
# TODO: install jq

info() {
    echo -en "\033[36m" ## blue
    echo $1
    echo -en "\033[0m" ## reset color
}

error() {
    echo -en "\033[31m" ## red
    echo $1
    echo -en "\033[0m" ## reset color
}

way=$(cd `dirname $0` && pwd)
num=`eval cat ${way}/AllImages.json | jq ".images | length"`
echo "total ${num} images needed to be build"

repo=linkerrepository

if [[ $num -ne 0 ]]
then
    for (( i = 0; i < $num; i++)); do
        image=`eval cat ${way}/AllImages.json | jq -r ".images[$i].name"`
        path=`eval cat ${way}/AllImages.json | jq -r ".images[$i].path"`
        dockerfile=`eval cat ${way}/AllImages.json | jq -r ".images[$i].dockerfile"`
        image_tag=${repo}/${image}
        context_path=${way}/${path}
        if [[ ${dockerfile} = null ]]; then
            dockerfile=Dockerfile
        fi
        echo "${image_tag}:"
        docker build -t ${image_tag} -f ${context_path}/${dockerfile} ${context_path} >/dev/null 2>&1
        result=$?
        if [ $result -ne 0 ]
        then
            error "---------- [build faild]"
        else
            info "---------- [build success]"
            docker push ${image_tag} > /dev/null 2>&1
            push_result=$?
            if [[ ${push_result} -ne 0 ]]; then
                error "---------- [push faild]"
            else
                info "---------- [push success]"
            fi
        fi
    done
fi
