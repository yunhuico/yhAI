package schedule

import (
	"errors"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

func init() {
	workflow.RegistryNodeMeta(&Timezone{})
}

// Timezone is a huge list, let's maintain it in one place, here.
type Timezone struct {
	// language that user prefers, e.g. zh-CN, en-US
	Language string `json:"X-Language"`
}

func (t *Timezone) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec(adapterClass.SpecClass("timezone"))
	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return new(Timezone)
		},
		InputForm: spec.InputSchema,
	}
}

func (t *Timezone) Run(c *workflow.NodeContext) (output any, err error) {
	err = errors.New("the adapter should not be reached")
	return
}

func (t *Timezone) QueryFieldResultList(c *workflow.NodeContext) (result workflow.QueryFieldResult, err error) {
	switch t.Language {
	case "zh-CN":
		result = workflow.QueryFieldResult{
			Items:  timezoneListZH,
			NoMore: true,
		}
	default:
		result = workflow.QueryFieldResult{
			Items:  timezoneListEN,
			NoMore: true,
		}
	}

	return
}

// timezoneListEN borrowed and adapted from
// https://raw.githubusercontent.com/dmfilipenko/timezones.json/master/timezones.json
var timezoneListEN = []workflow.QueryFieldItem{
	{Label: "(UTC+08:00) Beijing, Shanghai, Chongqing, Hong Kong, Urumqi, Taipei", Value: "Asia/Shanghai"},
	{Label: "(UTC-12:00) International Date Line West", Value: "Etc/GMT+12"},
	{Label: "(UTC-11:00) Midway, Niue, Pago Pago", Value: "Etc/GMT+11"},
	{Label: "(UTC-10:00) Hawaii", Value: "Etc/GMT+10"},
	{Label: "(UTC-09:00) Alaska", Value: "America/Anchorage"},
	{Label: "(UTC-08:00) Baja California", Value: "America/Santa_Isabel"},
	{Label: "(UTC-07:00) Pacific Standard Time (US & Canada)", Value: "America/Los_Angeles"},
	{Label: "(UTC-07:00) Arizona", Value: "America/Creston"},
	{Label: "(UTC-07:00) Chihuahua, La Paz, Mazatlan", Value: "America/Chihuahua"},
	{Label: "(UTC-07:00) Mountain Time (US & Canada)", Value: "America/Boise"},
	{Label: "(UTC-06:00) Central America", Value: "America/Belize"},
	{Label: "(UTC-06:00) Central Time (US & Canada)", Value: "America/Chicago"},
	{Label: "(UTC-06:00) Guadalajara, Mexico City, Monterrey", Value: "America/Bahia_Banderas"},
	{Label: "(UTC-06:00) Saskatchewan", Value: "America/Regina"},
	{Label: "(UTC-05:00) Bogota, Lima, Quito", Value: "America/Bogota"},
	{Label: "(UTC-05:00) Eastern Time (US & Canada)", Value: "America/Detroit"},
	{Label: "(UTC-05:00) Indiana (East)", Value: "America/Indiana/Marengo"},
	{Label: "(UTC-04:30) Caracas", Value: "America/Caracas"},
	{Label: "(UTC-04:00) Asuncion", Value: "America/Asuncion"},
	{Label: "(UTC-04:00) Atlantic Time (Canada)", Value: "America/Glace_Bay"},
	{Label: "(UTC-04:00) Cuiaba", Value: "America/Campo_Grande"},
	{Label: "(UTC-04:00) Georgetown, La Paz, Manaus, San Juan", Value: "America/Anguilla"},
	{Label: "(UTC-04:00) Santiago", Value: "America/Santiago"},
	{Label: "(UTC-03:30) Newfoundland", Value: "America/St_Johns"},
	{Label: "(UTC-03:00) Brasilia", Value: "America/Sao_Paulo"},
	{Label: "(UTC-03:00) Buenos Aires", Value: "America/Argentina/La_Rioja"},
	{Label: "(UTC-03:00) Cayenne, Fortaleza", Value: "America/Araguaina"},
	{Label: "(UTC-03:00) Greenland", Value: "America/Godthab"},
	{Label: "(UTC-03:00) Montevideo", Value: "America/Montevideo"},
	{Label: "(UTC-03:00) Salvador", Value: "America/Bahia"},
	{Label: "(UTC-02:00) Noronha, South Georgia", Value: "America/Noronha"},
	{Label: "(UTC-01:00) Azores", Value: "America/Scoresbysund"},
	{Label: "(UTC-01:00) Cape Verde Is.", Value: "Atlantic/Cape_Verde"},
	{Label: "(UTC) Casablanca", Value: "Africa/Casablanca"},
	{Label: "(UTC) Coordinated Universal Time", Value: "America/Danmarkshavn"},
	{Label: "(UTC) Edinburgh, London", Value: "Europe/Isle_of_Man"},
	{Label: "(UTC) Dublin, Lisbon", Value: "Atlantic/Canary"},
	{Label: "(UTC) Monrovia, Reykjavik", Value: "Africa/Abidjan"},
	{Label: "(UTC+01:00) Amsterdam, Berlin, Bern, Rome, Stockholm, Vienna", Value: "Arctic/Longyearbyen"},
	{Label: "(UTC+01:00) Belgrade, Bratislava, Budapest, Ljubljana, Prague", Value: "Europe/Belgrade"},
	{Label: "(UTC+01:00) Brussels, Copenhagen, Madrid, Paris", Value: "Africa/Ceuta"},
	{Label: "(UTC+01:00) Sarajevo, Skopje, Warsaw, Zagreb", Value: "Europe/Sarajevo"},
	{Label: "(UTC+01:00) West Central Africa", Value: "Africa/Algiers"},
	{Label: "(UTC+01:00) Windhoek", Value: "Africa/Windhoek"},
	{Label: "(UTC+02:00) Athens, Bucharest", Value: "Asia/Nicosia"},
	{Label: "(UTC+02:00) Beirut", Value: "Asia/Beirut"},
	{Label: "(UTC+02:00) Cairo", Value: "Africa/Cairo"},
	{Label: "(UTC+02:00) Damascus", Value: "Asia/Damascus"},
	{Label: "(UTC+02:00) Harare, Pretoria", Value: "Africa/Blantyre"},
	{Label: "(UTC+02:00) Helsinki, Kyiv, Riga, Sofia, Tallinn, Vilnius", Value: "Europe/Helsinki"},
	{Label: "(UTC+03:00) Istanbul", Value: "Europe/Istanbul"},
	{Label: "(UTC+02:00) Jerusalem", Value: "Asia/Jerusalem"},
	{Label: "(UTC+02:00) Tripoli", Value: "Africa/Tripoli"},
	{Label: "(UTC+03:00) Amman", Value: "Asia/Amman"},
	{Label: "(UTC+03:00) Baghdad", Value: "Asia/Baghdad"},
	{Label: "(UTC+02:00) Kaliningrad", Value: "Europe/Kaliningrad"},
	{Label: "(UTC+03:00) Kuwait, Riyadh", Value: "Asia/Aden"},
	{Label: "(UTC+03:00) Nairobi", Value: "Africa/Addis_Ababa"},
	{Label: "(UTC+03:00) Moscow, St. Petersburg, Volgograd, Minsk", Value: "Europe/Kirov"},
	{Label: "(UTC+04:00) Samara, Ulyanovsk, Saratov", Value: "Europe/Astrakhan"},
	{Label: "(UTC+03:30) Tehran", Value: "Asia/Tehran"},
	{Label: "(UTC+04:00) Abu Dhabi, Muscat", Value: "Asia/Dubai"},
	{Label: "(UTC+04:00) Baku", Value: "Asia/Baku"},
	{Label: "(UTC+04:00) Port Louis", Value: "Indian/Mahe"},
	{Label: "(UTC+04:00) Tbilisi", Value: "Asia/Tbilisi"},
	{Label: "(UTC+04:00) Yerevan", Value: "Asia/Yerevan"},
	{Label: "(UTC+04:30) Kabul", Value: "Asia/Kabul"},
	{Label: "(UTC+05:00) Ashgabat, Tashkent", Value: "Antarctica/Mawson"},
	{Label: "(UTC+05:00) Yekaterinburg", Value: "Asia/Yekaterinburg"},
	{Label: "(UTC+05:00) Islamabad, Karachi", Value: "Asia/Karachi"},
	{Label: "(UTC+05:30) Chennai, Kolkata, Mumbai, New Delhi", Value: "Asia/Kolkata"},
	{Label: "(UTC+05:30) Sri Jayawardenepura", Value: "Asia/Colombo"},
	{Label: "(UTC+05:45) Kathmandu", Value: "Asia/Kathmandu"},
	{Label: "(UTC+06:00) Nur-Sultan (Astana)", Value: "Antarctica/Vostok"},
	{Label: "(UTC+06:00) Dhaka", Value: "Asia/Dhaka"},
	{Label: "(UTC+06:30) Yangon (Rangoon)", Value: "Asia/Rangoon"},
	{Label: "(UTC+07:00) Bangkok, Hanoi, Jakarta", Value: "Antarctica/Davis"},
	{Label: "(UTC+07:00) Novosibirsk", Value: "Asia/Novokuznetsk"},
	{Label: "(UTC+08:00) Krasnoyarsk", Value: "Asia/Krasnoyarsk"},
	{Label: "(UTC+08:00) Kuala Lumpur, Singapore", Value: "Asia/Brunei"},
	{Label: "(UTC+08:00) Perth", Value: "Antarctica/Casey"},
	{Label: "(UTC+08:00) Ulaanbaatar", Value: "Asia/Choibalsan"},
	{Label: "(UTC+08:00) Irkutsk", Value: "Asia/Irkutsk"},
	{Label: "(UTC+09:00) Osaka, Sapporo, Tokyo", Value: "Asia/Dili"},
	{Label: "(UTC+09:00) Seoul", Value: "Asia/Pyongyang"},
	{Label: "(UTC+09:30) Adelaide", Value: "Australia/Adelaide"},
	{Label: "(UTC+09:30) Darwin", Value: "Australia/Darwin"},
	{Label: "(UTC+10:00) Brisbane", Value: "Australia/Brisbane"},
	{Label: "(UTC+10:00) Canberra, Melbourne, Sydney", Value: "Australia/Melbourne"},
	{Label: "(UTC+10:00) Guam, Port Moresby", Value: "Antarctica/DumontDUrville"},
	{Label: "(UTC+10:00) Hobart", Value: "Australia/Currie"},
	{Label: "(UTC+09:00) Yakutsk", Value: "Asia/Chita"},
	{Label: "(UTC+11:00) Solomon Is., New Caledonia", Value: "Antarctica/Macquarie"},
	{Label: "(UTC+11:00) Vladivostok", Value: "Asia/Sakhalin"},
	{Label: "(UTC+12:00) Auckland, Wellington", Value: "Antarctica/McMurdo"},
	{Label: "(UTC+12:00) Fiji", Value: "Pacific/Fiji"},
	{Label: "(UTC+12:00) Magadan", Value: "Asia/Anadyr"},
	{Label: "(UTC+13:00) Nuku'alofa", Value: "Etc/GMT-13"},
	{Label: "(UTC+13:00) Samoa", Value: "Pacific/Apia"},
}

var timezoneListZH = []workflow.QueryFieldItem{
	{Label: "(UTC+08:00) 北京、上海、重庆、香港、乌鲁木齐、台北", Value: "Asia/Shanghai"},
	{Label: "(UTC-12:00) 国际日期变更线西", Value: "Etc/GMT+12"},
	{Label: "(UTC-11:00) 中途岛、纽埃岛、帕果帕果", Value: "Etc/GMT+11"},
	{Label: "(UTC-10:00) 夏威夷", Value: "Etc/GMT+10"},
	{Label: "(UTC-09:00) 阿拉斯加", Value: "America/Anchorage"},
	{Label: "(UTC-08:00) 下加利福尼亚州", Value: "America/Santa_Isabel"},
	{Label: "(UTC-07:00) 太平洋标准时间（美国和加拿大）", Value: "America/Los_Angeles"},
	{Label: "(UTC-07:00) 亚利桑那", Value: "America/Creston"},
	{Label: "(UTC-07:00) 奇瓦瓦、拉巴斯、马萨特兰", Value: "America/Chihuahua"},
	{Label: "(UTC-07:00) 山区时间（美国和加拿大）", Value: "America/Boise"},
	{Label: "(UTC-06:00) 中美洲", Value: "America/Belize"},
	{Label: "(UTC-06:00) 中部时间（美国和加拿大）", Value: "America/Chicago"},
	{Label: "(UTC-06:00) 瓜达拉哈拉、墨西哥城、蒙特雷", Value: "America/Bahia_Banderas"},
	{Label: "(UTC-06:00) 萨斯喀彻温省", Value: "America/Regina"},
	{Label: "(UTC-05:00) 波哥大、利马、基多", Value: "America/Bogota"},
	{Label: "(UTC-05:00) 东部时间（美国和加拿大）", Value: "America/Detroit"},
	{Label: "(UTC-05:00) 印第安纳州（东部）", Value: "America/Indiana/Marengo"},
	{Label: "(UTC-04:30) 加拉加斯", Value: "America/Caracas"},
	{Label: "(UTC-04:00) 亚松森", Value: "America/Asuncion"},
	{Label: "(UTC-04:00) 大西洋时间（加拿大）", Value: "America/Glace_Bay"},
	{Label: "(UTC-04:00) 库亚巴", Value: "America/Campo_Grande"},
	{Label: "(UTC-04:00) 乔治敦、拉巴斯、马瑙斯、圣胡安", Value: "America/Anguilla"},
	{Label: "(UTC-04:00) 圣地亚哥", Value: "America/Santiago"},
	{Label: "(UTC-03:30) 纽芬兰", Value: "America/St_Johns"},
	{Label: "(UTC-03:00) 巴西利亚", Value: "America/Sao_Paulo"},
	{Label: "(UTC-03:00) 布宜诺斯艾利斯", Value: "America/Argentina/La_Rioja"},
	{Label: "(UTC-03:00) 卡宴、福塔莱萨", Value: "America/Araguaina"},
	{Label: "(UTC-03:00) 格陵兰", Value: "America/Godthab"},
	{Label: "(UTC-03:00) 蒙得维的亚", Value: "America/Montevideo"},
	{Label: "(UTC-03:00) 萨尔瓦多", Value: "America/Bahia"},
	{Label: "(UTC-02:00) 南乔治亚州诺罗尼亚", Value: "America/Noronha"},
	{Label: "(UTC-01:00) 亚速尔群岛", Value: "America/Scoresbysund"},
	{Label: "(UTC-01:00) 佛得角群岛。", Value: "Atlantic/Cape_Verde"},
	{Label: "(UTC) 卡萨布兰卡", Value: "Africa/Casablanca"},
	{Label: "(UTC) 协调世界时", Value: "America/Danmarkshavn"},
	{Label: "(UTC) 爱丁堡、伦敦", Value: "Europe/Isle_of_Man"},
	{Label: "(UTC) 都柏林、里斯本", Value: "Atlantic/Canary"},
	{Label: "(UTC) 蒙罗维亚、雷克雅未克", Value: "Africa/Abidjan"},
	{Label: "(UTC+01:00) 阿姆斯特丹、柏林、伯尔尼、罗马、斯德哥尔摩、维也纳", Value: "Arctic/Longyearbyen"},
	{Label: "(UTC+01:00) 贝尔格莱德、布拉迪斯拉发、布达佩斯、卢布尔雅那、布拉格", Value: "Europe/Belgrade"},
	{Label: "(UTC+01:00) 布鲁塞尔、哥本哈根、马德里、巴黎", Value: "Africa/Ceuta"},
	{Label: "(UTC+01:00) 萨拉热窝、斯科普里、华沙、萨格勒布", Value: "Europe/Sarajevo"},
	{Label: "(UTC+01:00) 中西部非洲", Value: "Africa/Algiers"},
	{Label: "(UTC+01:00) 温得和克", Value: "Africa/Windhoek"},
	{Label: "(UTC+02:00) 雅典、布加勒斯特", Value: "Asia/Nicosia"},
	{Label: "(UTC+02:00) 贝鲁特", Value: "Asia/Beirut"},
	{Label: "(UTC+02:00) 开罗", Value: "Africa/Cairo"},
	{Label: "(UTC+02:00) 大马士革", Value: "Asia/Damascus"},
	{Label: "(UTC+02:00) 比勒陀利亚哈拉雷", Value: "Africa/Blantyre"},
	{Label: "(UTC+02:00) 赫尔辛基、基辅、里加、索非亚、塔林、维尔纽斯", Value: "Europe/Helsinki"},
	{Label: "(UTC+03:00) 伊斯坦布尔", Value: "Europe/Istanbul"},
	{Label: "(UTC+02:00) 耶路撒冷", Value: "Asia/Jerusalem"},
	{Label: "(UTC+02:00) 的黎波里", Value: "Africa/Tripoli"},
	{Label: "(UTC+03:00) 安曼", Value: "Asia/Amman"},
	{Label: "(UTC+03:00) 巴格达", Value: "Asia/Baghdad"},
	{Label: "(UTC+02:00) 加里宁格勒", Value: "Europe/Kaliningrad"},
	{Label: "(UTC+03:00) 科威特、利雅得", Value: "Asia/Aden"},
	{Label: "(UTC+03:00) 内罗毕", Value: "Africa/Addis_Ababa"},
	{Label: "(UTC+03:00) 莫斯科、圣彼得堡、伏尔加格勒、明斯克", Value: "Europe/Kirov"},
	{Label: "(UTC+04:00) 萨马拉、乌里扬诺夫斯克、萨拉托夫", Value: "Europe/Astrakhan"},
	{Label: "(UTC+03:30) 德黑兰", Value: "Asia/Tehran"},
	{Label: "(UTC+04:00) 阿布扎比、马斯喀特", Value: "Asia/Dubai"},
	{Label: "(UTC+04:00) 巴库", Value: "Asia/Baku"},
	{Label: "(UTC+04:00) 路易港", Value: "Indian/Mahe"},
	{Label: "(UTC+04:00) 第比利斯", Value: "Asia/Tbilisi"},
	{Label: "(UTC+04:00) 埃里温", Value: "Asia/Yerevan"},
	{Label: "(UTC+04:30) 喀布尔", Value: "Asia/Kabul"},
	{Label: "(UTC+05:00) 阿什哈巴德、塔什干", Value: "Antarctica/Mawson"},
	{Label: "(UTC+05:00) 叶卡捷琳堡", Value: "Asia/Yekaterinburg"},
	{Label: "(UTC+05:00) 伊斯兰堡、卡拉奇", Value: "Asia/Karachi"},
	{Label: "(UTC+05:30) 钦奈、加尔各答、孟买、新德里", Value: "Asia/Kolkata"},
	{Label: "(UTC+05:30) 斯里贾亚瓦德纳普拉", Value: "Asia/Colombo"},
	{Label: "(UTC+05:45) 加德满都", Value: "Asia/Kathmandu"},
	{Label: "(UTC+06:00) 努尔苏丹（阿斯塔纳）", Value: "Antarctica/Vostok"},
	{Label: "(UTC+06:00) 达卡", Value: "Asia/Dhaka"},
	{Label: "(UTC+06:30) 仰光（仰光）", Value: "Asia/Rangoon"},
	{Label: "(UTC+07:00) 曼谷、河内、雅加达", Value: "Antarctica/Davis"},
	{Label: "(UTC+07:00) 新西伯利亚", Value: "Asia/Novokuznetsk"},
	{Label: "(UTC+08:00) 克拉斯诺亚尔斯克", Value: "Asia/Krasnoyarsk"},
	{Label: "(UTC+08:00) 吉隆坡、新加坡", Value: "Asia/Brunei"},
	{Label: "(UTC+08:00) 珀斯", Value: "Antarctica/Casey"},
	{Label: "(UTC+08:00) 乌兰巴托", Value: "Asia/Choibalsan"},
	{Label: "(UTC+08:00) 伊尔库茨克", Value: "Asia/Irkutsk"},
	{Label: "(UTC+09:00) 大阪、札幌、东京", Value: "Asia/Dili"},
	{Label: "(UTC+09:00) 首尔", Value: "Asia/Pyongyang"},
	{Label: "(UTC+09:30) 阿德莱德", Value: "Australia/Adelaide"},
	{Label: "(UTC+09:30) 达尔文", Value: "Australia/Darwin"},
	{Label: "(UTC+10:00) 布里斯班", Value: "Australia/Brisbane"},
	{Label: "(UTC+10:00) 堪培拉、墨尔本、悉尼", Value: "Australia/Melbourne"},
	{Label: "(UTC+10:00) 关岛、莫尔兹比港", Value: "Antarctica/DumontDUrville"},
	{Label: "(UTC+10:00) 霍巴特", Value: "Australia/Currie"},
	{Label: "(UTC+09:00) 雅库茨克", Value: "Asia/Chita"},
	{Label: "(UTC+11:00) 所罗门群岛、新喀里多尼亚", Value: "Antarctica/Macquarie"},
	{Label: "(UTC+11:00) 符拉迪沃斯托克", Value: "Asia/Sakhalin"},
	{Label: "(UTC+12:00) 奥克兰、惠灵顿", Value: "Antarctica/McMurdo"},
	{Label: "(UTC+12:00) 斐济", Value: "Pacific/Fiji"},
	{Label: "(UTC+12:00) 马加丹", Value: "Asia/Anadyr"},
	{Label: "(UTC+13:00) 努库阿洛法", Value: "Etc/GMT-13"},
	{Label: "(UTC+13:00) 萨摩亚", Value: "Pacific/Apia"},
}
