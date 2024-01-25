package utils

import (
	"fmt"
	"path"
	"testing"
)

func TestEqualFloat64(t *testing.T) {
	type args struct {
		f1 interface{}
		f2 interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{"TestEqual", args{52, "51"}, 1, false},
		{"one", args{52, "51.5"}, 1, false},
		{"one", args{52.01, "51"}, 1, false},
		{"one", args{52.01, "51.5"}, 1, false},
		{"one", args{"52", "51"}, 1, false},
		{"one", args{"52", "51.5"}, 1, false},
		{"one", args{"52.5", "51"}, 1, false},
		{"one", args{"52.5", "51.5"}, 1, false},

		{"one", args{52, "52"}, 0, false},
		{"one", args{52, "52.00"}, 0, false},
		{"one", args{52.00, "52"}, 0, false},
		{"one", args{52.00, "52.00"}, 0, false},

		{"one", args{51, "52"}, -1, false},
		{"one", args{51, "52.5"}, -1, false},
		{"one", args{51.5, "52"}, -1, false},
		{"one", args{51.5, "52.5"}, -1, false},
		{"one", args{"51.5", "52"}, -1, false},
		{"one", args{"51.5", "52.5"}, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EqualFloat64(tt.args.f1, tt.args.f2)
			if (err != nil) != tt.wantErr {
				t.Errorf("EqualFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EqualFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatJSONStr(t *testing.T) {
	s := `{ NBS: [ "ChangBaiShanJiChang", "长白山机场", "NBSJC", "NBS", "中国", "changbaishanjichang", "", "NBS", "0", "机场", "长白山机场", ], HND: [ "DongJing", "东京(羽田)", "Dongjing|Yutian|yt|dj|日本|浅草寺|筑地市场|东京塔|riben|qiancaosi|zhudishichang|dongjingta", "HND", "日本", "Tokyo(Haneda)", "", "HND", "0", "东京羽田国际机场", "东京羽田", ], ICN: [ "shoueryunchuanjichang", "首尔仁川机场", "shoueryunchuanjichang", "ICN", "韩国", "shoueryunchuanjichang", "", "ICN", "0", "仁川国际机场", "首尔仁川", ], SEL: [ "Seoul", "首尔", "SE|ShouEr|Seoul|韩国|景福宫|乐天世界|首尔塔|hanguo|jingfugong|letianshijie|shouerta", "SEL", "韩国", "Seoul", "", "SEL", "0", "仁川国际机场", "首尔仁川", ], KOJ: [ "LuErDao", "鹿儿岛", "LED|Luerdao", "KOJ", "日本", "Kagoshima", "", "KOJ", "0", "机场", "鹿儿岛机场", ], YGJ: [ "NiaoQu", "鸟取", "NQ|Tottori|niaoqu|日本|riben", "YGJ", "日本", "Tottori", "", "YGJ", "0", "米子鬼太郎空港", "鸟取米子鬼太郎空港", ], CTS: [ "ZhaHuangJichang", "札幌机场", "zhahuangjichang", "CTS", "日本", "zhahuangjichang", "", "CTS", "0", "新千岁机场", "札幌新千岁机场", ], IBR: [ "CiChengJiChang", "茨城(近东京)机场", "cichengjichang", "IBR", "日本", "cichengjichang", "", "IBR", "0", "机场", "茨城(近东京)机场", ], TAK: [ "GaoSong", "高松", "gs|gaosong|日本|栗林公园|濑户内海|riben|lilingongyuan|laihuneihai", "TAK", "日本", "Takamatsu", "", "TAK", "0", "机场", "高松机场", ], MFM: [ "AoMen", "澳门(中国澳门)", "MC|aomen|中国|大三巴牌坊|渔人码头|妈祖阁|zhongguo|dasanbapaifang|yurenmatou|mazuge", "MFM", "中国澳门", "Macau", "", "MFM", "0", "澳门国际机场", "澳门", ], HSG: [ "Zuohe", "佐贺(近福冈)", "SAGA|zuohe|zh|fugang|日本|嬉野温泉|riben|xiyewenquan", "HSG", "日本", "Saga(Fukuoka)", "", "HSG", "0", "佐贺机场", "佐贺机场", ], BKK: [ "Mangu", "曼谷", "MG|Bangkok|mangu|泰国|大皇宫|玉佛寺|湄南河|taiguo|dahuanggong|yufosi|meinanhe", "BKK", "泰国", "Bangkok", "", "BKK", "0", "素万那普国际机场", "曼谷素万那普", ], HKT: [ "PuJi", "普吉", "PJ|puji|泰国|皇帝岛|芭东海滩|taiguo|huangdidao|badonghaitan", "HKT", "泰国", "Phuket", "", "HKT", "0", "国际机场", "普吉", ], REP: [ "XianLi", "暹粒", "XL|xianli|柬埔寨|吴哥窟|崩密列|女王宫|jianpuzhai|wugeku|bengmilie|nvwanggong", "REP", "柬埔寨", "Siem Reap", "", "REP", "0", "吴哥国际机场", "暹粒吴哥", ], CJU: [ "JiZhou", "济州岛", "JZ|jizhoudao|韩国|汉拿山|牛岛|泰迪熊博物馆|hanguo|hannashan|niudao|taidixiongbowuguan", "CJU", "韩国", "Jeju", "", "CJU", "0", "济州国际机场", "济州", ], BKI: [ "YaBi", "亚庇(哥打基纳巴卢)", "SB|shaba|马来西亚|卡帕莱岛|环滩岛|malaixiya|kapalaidao|huantandao|沙巴|亚庇|哥打京那巴鲁|Sabah|Kota Kinabalu|Jesselton", "BKI", "马来西亚", "Kota Kinabalu", "", "BKI", "0", "哥打基纳巴卢国际机场", "亚庇(哥打基纳巴卢)哥打基纳巴卢", ], OSA: [ "DaBan", "大阪", "DB|daban|Osaka|大阪|日本|日本环球影城|道顿堀|riben|ribenhuanqiuyingcheng|daodunku", "OSA", "日本", "Osaka", "", "OSA", "0", "关西国际机场", "大阪关西", ], HIJ: [ "GuangDao", "广岛", "GD|Guangdao|日本|riben", "HIJ", "日本", "Hiroshima", "", "HIJ", "0", "机场", "广岛机场", ], HKD: [ "HanGuan", "函馆", "HG|hanguan", "HKD", "日本", "Hakodate", "", "HKD", "0", "国际机场", "函馆", ], KMJ: [ "XiongBen", "熊本", "XB|xiongben", "KMJ", "日本", "Kumamoto", "", "KMJ", "0", "机场", "熊本机场", ], CNX: [ "QingMai", "清迈", "QM|qingmai|泰国|清迈夜市|清迈古城|taiguo|qingmaiyeshi|qingmaigucheng", "CNX", "泰国", "Chiang Mai", "", "CNX", "0", "国际机场", "清迈", ], DAD: [ "XianGang", "岘港", "XG|xiangang|越南|yuenan", "DAD", "越南", "DaNang", "", "DAD", "0", "国际机场", "岘港", ], SIN: [ "XinJiaPo", "新加坡", "XJP|Singapore|xinjiapo|鱼尾狮公园|圣淘沙岛|小印度|yuweishigongyuan|shengtaoshadao|xiaoyindu", "SIN", "新加坡", "Singapore", "", "SIN", "0", "樟宜机场", "新加坡樟宜机场", ], SPK: [ "ZhaHuang", "札幌", "ZH|ZhaHuang|Sapporo|日本|北海道|大通公园|riben|beihaidao|datonggongyuan", "SPK", "日本", "Sapporo", "", "SPK", "0", "新千岁机场", "札幌新千岁机场", ], KBV: [ "JiaMi", "甲米", "Jiami|JM|泰国|皮皮岛|玛雅湾|taiguo|pipidao|mayawan", "KBV", "泰国", "Krabi", "", "KBV", "0", "国际机场", "甲米", ], AKJ: [ "XuChuan", "旭川", "Xuchuan|xc|riben|日本", "AKJ", "日本", "Asahikawa", "", "AKJ", "0", "空港", "旭川空港", ], NGO: [ "MingGuWu", "名古屋", "mgw|mingguwu|日本|热田神宫|riben|retianshengong", "NGO", "日本", "Nagoya", "", "NGO", "0", "日本中部国际机场", "名古屋日本中部", ], PNH: [ "JinBian", "金边", "JinBian|PNH|jb|柬埔寨|金边皇宫|中央市场|柬埔寨国家博物馆|jianpuzhai|jinbianhuanggong|zhongyangshichang|jianpuzhaiguojiabowuguan", "PNH", "柬埔寨", "Phnom Penh", "", "PNH", "0", "国际机场", "金边", ], URT: [ "SuLeTaNi", "素叻他尼(近苏梅岛)", "Surat Thani|suletani|泰国|查汶海滩|taiguo|sumeidao|chawenhaitan", "URT", "泰国", "Surat Thani", "", "URT", "0", "素叻他尼(万伦)机场", "素叻他尼(万伦)机场", ], KUL: [ "JiLongPo", "吉隆坡", "Kuala Lumpu|jlp|jilongpo|KUL", "KUL", "马来西亚", "Kuala Lumpur", "", "KUL", "0", "国际机场", "吉隆坡吉隆坡", ], JHB: [ "XinShan", "新山", "Johor Bahru|xs|JHB|xinshan|马来西亚|malaixiya", "JHB", "马来西亚", "Johor Bahru", "", "JHB", "0", "士乃机场", "新山士乃机场", ], USM: [ "SuMeiDAo", "苏梅岛", "Samui|KS|USM|Sumeidao|smd", "USM", "泰国", "Koh Samui", "", "USM", "0", "机场", "苏梅岛机场", ], SGN: [ "HuZhiMingShi", "胡志明市", "HoChiMinhCity|HZM|huzhiming|越南|yuenan", "SGN", "越南", "Ho Chi Ming City", "", "SGN", "0", "新山一国际机场", "胡志明市新山一", ], TYO: [ "CiCheng", "茨城(近东京)", "IBRTYO|cicheng|cc|dongjing|dj|日本|筑波山|riben|zhuboshan", "TYO", "日本", "Ibaraki(Tokyo)", "", "TYO", "0", "机场", "茨城(近东京)机场", ], MAD: [ "Madrid", "马德里", "MDL|馬德里|Madrid", "MAD", "西班牙", "Madrid", "", "MAD", "0", "巴拉哈斯机场T4S", "巴拉哈斯机场T4S", ], KIX: [ "DaBanJiChang", "关西机场", "dabanjichang", "KIX", "日本", "dabanjichang", "", "KIX", "0", "关西国际机场", "大阪关西", ], KHH: [ "GaoXiong", "高雄(中国台湾)", "GX|Gaoxiong|中国|西子湾|旗津半岛|zhongguo|xiziwan|qijinbandao", "KHH", "中国台湾", "Kaohsiung", "", "KHH", "0", "中国台湾高雄国际机场", "中国台湾高雄", ], TSN: [ "Tianjin", "天津", "TJ|中国|天津之眼|五大道|zhongguo|tianjinzhiyan|wudadao", "TSN", "中国", "Tianjin", "", "TSN", "0", "滨海国际机场", "天津滨海", ], WUH: [ "Wuhan", "武汉", "WH|中国|黄鹤楼|zhongguo|huanghelou", "WUH", "中国", "Wuhan", "", "WUH", "0", "天河国际机场", "武汉天河", ], NGB: [ "NingBo", "宁波", "NB|中国|天一阁|渔山列岛|象山影视城|zhongguo|tianyige|yushanliedao|xiangshanyingshicheng", "NGB", "中国", "Ningbo", "", "NGB", "0", "栎社国际机场", "宁波栎社", ], TYN: [ "TaiYuan", "太原", "TY", "TYN", "中国", "Taiyuan", "", "TYN", "0", "武宿国际机场", "太原武宿", ], XUZ: [ "XuZhou", "徐州", "XZ", "XUZ", "中国", "Xuzhou", "", "XUZ", "0", "观音机场", "徐州观音机场", ], XFN: [ "Xiangyang", "襄阳", "XY", "XFN", "中国", "Xiangyang", "", "XFN", "0", "刘集机场", "襄阳刘集机场", ], LJG: [ "LiJiang", "丽江", "LJ", "LJG", "中国", "Lijiang", "", "LJG", "0", "三义机场", "丽江三义机场", ], LXA: [ "LaSa", "拉萨", "LS", "LXA", "中国", "Lhasa", "", "LXA", "0", "贡嘎国际机场", "拉萨贡嘎", ], NTG: [ "NanTong", "南通", "NT", "NTG", "中国", "Nantong", "", "NTG", "0", "兴东国际机场", "南通兴东", ], WXN: [ "WanZhou", "万州", "WZ", "WXN", "中国", "Wanzhou", "", "WXN", "0", "五桥机场", "万州五桥机场", ], XNN: [ "XiNing", "西宁", "XN", "XNN", "中国", "Xining", "", "XNN", "0", "曹家堡机场", "西宁曹家堡机场", ], YIH: [ "YiChang", "宜昌", "YC", "YIH", "中国", "Yichang", "", "YIH", "0", "三峡机场", "宜昌三峡机场", ], XIC: [ "XiChang", "西昌", "XC", "XIC", "中国", "Xichang", "", "XIC", "0", "青山机场", "西昌青山机场", ], TGO: [ "Tongliao", "通辽", "TL", "TGO", "中国", "Tongliao", "", "TGO", "0", "机场", "通辽机场", ], ZJA: [ "Zhenjiang", "镇江", "ZJ", "ZJA", "中国", "Zhengjiang", "", "ZJA", "0", "机场", "镇江机场", ], TOX: [ "Tongxiang", "桐乡", "TX", "TOX", "中国", "Tongxiang", "", "TOX", "0", "乌镇机场", "桐乡桐乡乌镇机场", ], SHX: [ "Shaoxing", "绍兴", "SX", "SHX", "中国", "Shaoxing", "", "SHX", "0", "机场", "绍兴机场", ], YIW: [ "Yiwu", "义乌", "YW", "YIW", "中国", "Yiwu", "", "YIW", "0", "机场", "义乌机场", ], KUS: [ "Kunshan", "昆山", "KS", "KUS", "中国", "Kunshan", "", "KUS", "0", "机场", "昆山机场", ], LZO: [ "Luzhou", "泸州", "LZ", "LZO", "中国", "Luzhou", "", "LZO", "0", "蓝田机场", "泸州蓝田机场", ], LZH: [ "LiuZhou", "柳州", "LZ", "LZH", "中国", "Liuzhou", "", "LZH", "0", "白莲机场", "柳州白莲机场", ], TEN: [ "Tongren", "铜仁", "TR", "TEN", "中国", "Tongren", "", "TEN", "0", "凤凰机场", "铜仁凤凰机场", ], YNT: [ "YanTai", "烟台", "YT", "YNT", "中国", "Yantai", "", "YNT", "0", "莱山国际机场", "烟台莱山", ], TNA: [ "JiNan", "济南", "JN", "TNA", "中国", "Jinan", "", "TNA", "0", "遥墙国际机场", "济南遥墙", ], SHP: [ "QinHuangDao", "秦皇岛", "QHD", "SHP", "中国", "Qinhuangdao", "", "SHP", "0", "山海关机场", "秦皇岛山海关机场", ], KOW: [ "GanZhou", "赣州", "GZ", "KOW", "中国", "Ganzhou", "", "KOW", "0", "黄金机场", "赣州黄金机场", ], TVS: [ "TangShan", "唐山", "TS", "TVS", "中国", "Tangshan", "", "TVS", "0", "三女河机场", "唐山三女河机场", ], TCZ: [ "TengChong", "腾冲", "TC", "TCZ", "中国", "Tengchong", "", "TCZ", "0", "驼峰机场", "腾冲驼峰机场", ], YCU: [ "YunCheng", "运城", "YC", "YCU", "中国", "Yuncheng", "", "YCU", "0", "关公机场", "运城关公机场", ], WUZ: [ "WuZhou", "梧州", "WZ", "WUZ", "中国", "Wuzhou", "", "WUZ", "0", "长洲岛机场", "梧州长洲岛机场", ], YNJ: [ "YanJi", "延吉", "YJ", "YNJ", "中国", "Yanji", "", "YNJ", "0", "朝阳川国际机场", "延吉朝阳川", ], NZH: [ "Manzhouli", "满洲里", "MZL", "NZH", "中国", "Manzhouli", "", "NZH", "0", "西郊机场", "满洲里西郊机场", ], ZQZ: [ "ZhangJiaKou", "张家口", "ZJK", "ZQZ", "中国", "Zhangjiakou", "", "ZQZ", "0", "机场", "张家口机场", ], PEK: [ "BeiJing", "北京", "BJ|中国|故宫|颐和园|天安门广场|zhongguo|gugong|yiheyuan|tiananmenguangchang", "PEK", "中国", "Beijing", "", "PEK", "0", "首都国际机场", "北京首都", ], YNZ: [ "YanCheng", "盐城", "Yancheng|YC", "YNZ", "中国", "Yancheng", "", "YNZ", "0", "南洋国际机场", "盐城南洋", ], YZY: [ "ZhangYe", "张掖", "zy|zhangye", "YZY", "中国", "Zhangye", "", "YZY", "0", "甘州机场", "张掖甘州机场", ], KJH: [ "KaiLi", "凯里", "Kaili|kl", "KJH", "中国", "Kaili", "", "KJH", "0", "黄平机场", "凯里黄平机场", ], WUR: [ "WuShi", "乌什", "WS", "WUR", "中国", "Wushi", "", "WUR", "0", "烏什机场", "烏什机场", ], XJS: [ "XinJi", "辛集", "XINJI|xj|XJS|xinji", "XJS", "中国", "Xinji", "", "XJS", "0", "机场", "辛集机场", ], SQJ: [ "SanMing", "三明", "SM", "SQJ", "中国", "Sanming", "", "SQJ", "0", "SQJ", "SQJ", ], NNY: [ "NanYang", "南阳", "NanYang|ny", "NNY", "中国", "NanYang", "", "NNY", "0", "姜营机场", "南阳姜营机场", ], LHZ: [ "LinHai", "临海", "linhai|LH", "LHZ", "中国", "Linhai", "", "LHZ", "0", "火车站", "临海火车站", ], QSZ: [ "ShaChe", "莎车", "Shache|SC", "QSZ", "中国", "Shache", "", "QSZ", "0", "叶尔羌机场", "莎车叶尔羌机场", ], WGN: [ "ShaoYang", "邵阳", "Shaoyang|SY|Wugang|邵阳|武冈|武岡|邵陽", "WGN", "中国", "Shaoyang", "", "WGN", "0", "武冈机场", "邵阳武冈机场", ], SJW: [ "ShiJiaZhuang", "石家庄", "SJZ", "SJW", "中国", "Shijiazhuang", "", "SJW", "0", "正定国际机场", "石家庄正定", ], SHE: [ "ShenYang", "沈阳", "SY", "SHE", "中国", "Shenyang", "", "SHE", "0", "桃仙国际机场", "沈阳桃仙", ], YTY: [ "YangZhou", "扬州(泰州)", "Yangzhou|yz|taizhou|tz|中国|瘦西湖|文昌阁|zhongguo|shouxihu|wenchangge|扬泰|杨泰|yangtai|杨洲|杨州|杨州|泰洲|关东街", "YTY", "中国", "Yangzhou", "", "YTY", "0", "扬州泰州国际机场", "扬州泰州", ], XMN: [ "XiaMen", "厦门", "XM|中国|鼓浪屿|南普陀寺|曾厝垵|zhongguo|gulangyu|nanputuosi|zengcuoan", "XMN", "中国", "Xiamen", "", "XMN", "0", "高崎国际机场", "厦门高崎", ], LHW: [ "LanZhou", "兰州", "LZ", "LHW", "中国", "Lanzhou", "", "LHW", "0", "中川机场", "兰州中川机场", ], SYX: [ "SanYa", "三亚", "SY|中国|天涯海角|亚龙湾|蜈支洲岛|zhongguo|tianyahaijiao|yalongwan|wuzhizhoudao", "SYX", "中国", "Sanya", "", "SYX", "0", "凤凰国际机场", "三亚凤凰", ], SWA: [ "JieYang", "揭阳(潮汕)", "JY|shantou|st|chaoshan|chaozhou|cs|cz", "SWA", "中国", "Jieyang(Chaoshan)", "", "SWA", "0", "揭阳潮汕机场", "揭阳潮汕机场", ], ZHA: [ "ZhanJiang", "湛江", "ZJ", "ZHA", "中国", "Zhanjiang", "", "ZHA", "0", "机场", "湛江机场", ], MIG: [ "MianYang", "绵阳", "MY", "MIG", "中国", "Mianyang", "", "MIG", "0", "南郊机场", "绵阳南郊机场", ], KWE: [ "Guiyang", "贵阳", "GY|中国|青岩古镇|花溪公园|zhongguo|qingyanguzhen|huaxigongyuan", "KWE", "中国", "Guiyang", "", "KWE", "0", "龙洞堡国际机场", "贵阳龙洞堡", ], NNG: [ "NanNing", "南宁", "NN", "NNG", "中国", "Nanning", "", "NNG", "0", "吴圩国际机场", "南宁吴圩", ], ZUH: [ "ZhuHai", "珠海", "ZH", "ZUH", "中国", "Zhuhai", "", "ZUH", "0", "金湾机场", "珠海金湾机场", ], URC: [ "WuLuMuQi", "乌鲁木齐", "WLMQ|中国|天山大峡谷|盐湖|zhongguo|tianshandaxiagu|yanhu", "URC", "中国", "Urumqi", "", "URC", "0", "地窝堡国际机场", "乌鲁木齐地窝堡", ], LYA: [ "LuoYang", "洛阳", "LY", "LYA", "中国", "Luoyang", "", "LYA", "0", "北郊机场", "洛阳北郊机场", ], SZX: [ "ShenZhen", "深圳", "SZ|中国|世界之窗|大梅沙|莲花山|zhongguo|shijiezhichuang|dameisha|lianhuashan", "SZX", "中国", "Shenzhen", "", "SZX", "0", "宝安国际机场", "深圳宝安", ], TAO: [ "QingDao", "青岛", "QD|中国|八大关|崂山|栈桥|zhongguo|badaguan|laoshan|zhanqiao", "TAO", "中国", "Qingdao", "", "TAO", "0", "流亭国际机场", "青岛流亭", ], KWL: [ "GuiLin", "桂林", "GL|中国|漓江|象山公园|阳朔|zhongguo|lijiang|xiangshangongyuan|yangshuo", "KWL", "中国", "Guilin", "", "KWL", "0", "两江国际机场", "桂林两江", ], NKG: [ "NanJing", "南京", "NJ|中国|夫子庙|玄武湖|栖霞山|zhongguo|fuzimiao|xuanwuhu|qixiashan", "NKG", "中国", "Nanjing", "", "NKG", "0", "禄口国际机场", "南京禄口", ], WUX: [ "WuXi", "无锡", "WX|中国|鼋头渚|宜兴竹海|zhongguo|yuantouzhu|yixingzhuhai", "WUX", "中国", "Wuxi", "", "WUX", "0", "苏南硕放国际机场", "无锡苏南硕放", ], SZV: [ "SuZhou", "苏州", "SZ|中国|拙政园|平江路|周庄古镇|zhongguo|zhuozhengyuan|pingjianglu|zhouzhuangguzhen", "SZV", "中国", "Suzhou", "", "SZV", "0", "机场", "苏州机场", ], XNT: [ "XingTai", "邢台", "XNT|xt|Xingtai", "XNT", "中国", "Xingtai", "", "XNT", "0", "机场", "邢台机场", ], KHN: [ "NanChang", "南昌", "NC", "KHN", "中国", "Nanchang", "", "KHN", "0", "昌北国际机场", "南昌昌北", ], JZH: [ "JiuZhaiGou", "九寨沟", "JZG", "JZH", "中国", "Jiuzhaigou", "", "JZH", "0", "黄龙机场", "九寨沟黄龙机场", ], JNG: [ "JiNing", "济宁", "JN", "JNG", "中国", "Jining", "", "JNG", "0", "曲阜机场", "济宁曲阜机场", ], JJN: [ "QuanZhou", "泉州(晋江)", "JJ|QZ|jinjiang", "JJN", "中国", "Quanzhou(Jinjiang)", "", "JJN", "0", "泉州晋江机场", "泉州晋江机场", ], JIX: [ "JiaXing", "嘉兴", "JX", "JIX", "中国", "Jiaxing", "", "JIX", "0", "机场", "嘉兴机场", ], JIQ: [ "QianJiang", "黔江", "QJ", "JIQ", "中国", "Qianjiang", "", "JIQ", "0", "武陵山机场", "黔江武陵山机场", ], JHG: [ "Xishuangbanna", "西双版纳", "XSBN|Xishuangbanna", "JHG", "中国", "Xishuangbanna", "", "JHG", "0", "嘎洒国际机场", "西双版纳嘎洒", ], JGS: [ "JingGangShan", "井冈山", "JGS|Jinggangshan|Jingangshan|Jingganshan", "JGS", "中国", "Jinggangshan", "", "JGS", "0", "机场", "井冈山机场", ], IQN: [ "Qingyang", "庆阳", "QY", "IQN", "中国", "Qingyang", "", "IQN", "0", "机场", "庆阳机场", ], INC: [ "YinChuan", "银川", "YC", "INC", "中国", "Yinchuan", "", "INC", "0", "河东机场", "银川河东机场", ], HZH: [ "LiPing", "黎平", "LP", "HZH", "中国", "Liping", "", "HZH", "0", "机场", "黎平机场", ], HYN: [ "TaiZhou", "台州", "TZ", "HYN", "中国", "Taizhou", "", "HYN", "0", "路桥机场", "台州路桥机场", ], HUN: [ "HuaLian", "花莲", "Hualian|hl", "HUN", "中国", "Hualian", "", "HUN", "0", "机场", "花莲机场", ], HSN: [ "ZhouShan", "舟山", "ZS", "HSN", "中国", "Zhoushan", "", "HSN", "0", "普陀山机场", "舟山普陀山机场", ], HRB: [ "HaErBin", "哈尔滨", "HEB|中国|冰雪大世界|中央大街|亚布力|zhongguo|bingxuedashijie|zhongyangdajie|yabuli", "HRB", "中国", "Harbin", "", "HRB", "0", "太平国际机场", "哈尔滨太平", ], HLD: [ "HaiLaEr", "海拉尔", "HLE", "HLD", "中国", "Hailaer", "", "HLD", "0", "东山机场", "海拉尔东山机场", ], HJJ: [ "HuaiHua", "怀化", "HH", "HJJ", "中国", "Huaihua", "", "HJJ", "0", "芷江机场", "怀化芷江机场", ], HIA: [ "Huaian", "淮安", "HA", "HIA", "中国", "Huaian", "", "HIA", "0", "涟水机场", "淮安涟水机场", ], HGH: [ "HangZhou", "杭州", "HZ|中国|西湖|灵隐寺|千岛湖|zhongguo|xihu|lingyinsi|qiandaohu", "HGH", "中国", "Hangzhou", "", "HGH", "0", "萧山国际机场", "杭州萧山", ], HFE: [ "HeFei", "合肥", "HF", "HFE", "中国", "Hefei", "", "HFE", "0", "新桥国际机场", "合肥新桥", ], HET: [ "HuHeHaoTe", "呼和浩特", "HHHT", "HET", "中国", "Hohhot", "", "HET", "0", "白塔国际机场", "呼和浩特白塔", ], HEK: [ "HeiHe", "黑河", "HH", "HEK", "中国", "Heihe", "", "HEK", "0", "机场", "黑河机场", ], HDG: [ "HanDan", "邯郸", "HD", "HDG", "中国", "Handan", "", "HDG", "0", "机场", "邯郸机场", ], WNZ: [ "WenZhou", "温州", "WZ", "WNZ", "中国", "Wenzhou", "", "WNZ", "0", "龙湾国际机场", "温州龙湾", ], HSU: [ "HengShui", "衡水", "HSU|hs|Hengshui", "HSU", "中国", "Hengshui", "", "HSU", "0", "机场", "衡水机场", ], ACX: [ "XingYi", "兴义", "XY", "ACX", "中国", "Xingyi", "", "ACX", "0", "机场", "兴义机场", ], AEB: [ "BaiSe", "百色", "BS", "AEB", "中国", "Baise", "", "AEB", "0", "机场", "百色机场", ], AGS: [ "AnGuo", "安国", "ANGUO|ag|AGS|anguo", "AGS", "中国", "Anguo", "", "AGS", "0", "机场", "安国机场", ], AVA: [ "AnShun", "安顺(黄果树机场)", "Anshun|AS|huang|huangg|huangguo|huangguoshu|hgs|中国|天星桥|zhongguo|huangguoshupubu|tianxingqiao", "AVA", "中国", "Anshun", "", "AVA", "0", "安顺黄果树机场", "安顺黄果树机场", ], BAD: [ "BaoDing", "保定", "BAD|bd|Baoding", "BAD", "中国", "Baoding", "", "BAD", "0", "机场", "保定机场", ], BAS: [ "ChangBaiShan", "长白山", "CBS|bs|bais|baishan", "BAS", "中国", "Changbaishan", "", "BAS", "0", "机场", "长白山机场", ], BFJ: [ "BiJie", "毕节", "Bijie|BJ", "BFJ", "中国", "Bijie", "", "BFJ", "0", "飞雄机场", "毕节飞雄机场", ], BHY: [ "BeiHai", "北海", "BH", "BHY", "中国", "Beihai", "", "BHY", "0", "福成机场", "北海福成机场", ], CDE: [ "ChengDe", "承德", "Chengde|CD", "CDE", "中国", "Chengde", "", "CDE", "0", "普宁机场", "承德普宁机场", ], CGD: [ "ChangDe", "常德", "CD", "CGD", "中国", "Changde", "", "CGD", "0", "桃花源机场", "常德桃花源机场", ], CAN: [ "GuangZhou", "广州", "GZ|中国|白云山|中山纪念堂|zhongguo|baiyunshan|zhongshanjiniantang", "CAN", "中国", "Guangzhou", "", "CAN", "0", "白云国际机场", "广州白云", ], CGO: [ "ZhengZhou", "郑州", "ZZ", "CGO", "中国", "Zhengzhou", "", "CGO", "0", "新郑国际机场", "郑州新郑", ], CGQ: [ "ChangChun", "长春", "CC", "CGQ", "中国", "Changchun", "", "CGQ", "0", "龙嘉国际机场", "长春龙嘉", ], CHG: [ "ChaoYang", "朝阳", "CY", "CHG", "中国", "Chaoyang", "", "CHG", "0", "机场", "朝阳机场", ], CZX: [ "ChangZhou", "常州", "CZ", "CZX", "中国", "Changzhou", "", "CZX", "0", "奔牛机场", "奔牛机场", ], CSX: [ "ChangSha", "长沙", "CS", "CSX", "中国", "Changsha", "", "CSX", "0", "黄花国际机场", "长沙黄花", ], CTU: [ "ChengDu", "成都", "CD|中国|宽窄巷子|大熊猫基地|都江堰|zhongguo|kuanzhaixiangzi|daxiongmaojidi|dujiangyan", "CTU", "中国", "Chengdu", "", "CTU", "0", "双流国际机场", "成都双流", ], CYI: [ "JiaYi", "嘉义", "JY|JiaYi", "CYI", "中国台湾", "Chiayi", "", "CYI", "0", "水上机场", "嘉义水上机场", ], DAT: [ "DaTong", "大同", "DT", "DAT", "中国", "Datong", "", "DAT", "0", "山西机场", "山西大同机场", ], DIG: [ "DiQing", "迪庆", "DQ", "DIG", "中国", "Diqing", "", "DIG", "0", "机场", "迪庆机场", ], DLC: [ "DaLian", "大连", "DL|中国|星海广场|老虎滩|zhongguo|xinghaiguangchang|laohutan", "DLC", "中国", "Dalian", "", "DLC", "0", "周水子国际机场", "大连周水子", ], DNH: [ "DunHuang", "敦煌", "DNH|dh|Dunhuang|中国|莫高窟|鸣沙山|雅丹|zhongguo|mogaoku|mingshashan|yadan", "DNH", "中国", "Dunhuang", "", "DNH", "0", "机场", "敦煌机场", ], DOY: [ "DongYing", "东营", "DY", "DOY", "中国", "Dongying", "", "DOY", "0", "胜利机场", "东营胜利机场", ], DYA: [ "Danyang", "丹阳", "DY", "DYA", "中国", "Danyang", "", "DYA", "0", "机场", "丹阳机场", ], DYG: [ "ZhangJiaJie", "张家界", "ZJJ|中国|天门山|黄龙洞|武陵源|zhongguo|tianmenshan|huanglongdong|wulingyuan", "DYG", "中国", "Zhangjiajie", "", "DYG", "0", "荷花国际机场", "张家界荷花", ], DZS: [ "DingZhou", "定州", "DINGZHOU|dz|DZS|dingzhou", "DZS", "中国", "Dingzhou", "", "DZS", "0", "机场", "定州机场", ], ENH: [ "EnShi", "恩施", "Enshi|es", "ENH", "中国", "Enshi", "", "ENH", "0", "许家坪机场", "恩施许家坪机场", ], FOC: [ "FuZhou", "福州", "FZ|中国|三坊七巷|鼓山|zhongguo|sanfangqixiang|gushan", "FOC", "中国", "Fuzhou", "", "FOC", "0", "长乐国际机场", "福州长乐", ], GGN: [ "QuanZhou", "泉州", "QZ", "GGN", "中国", "Quanzhou", "", "GGN", "0", "晋江国际机场", "泉州晋江", ], HAK: [ "HaiKou", "海口", "HK", "HAK", "中国", "Haikou", "", "HAK", "0", "美兰国际机场", "海口美兰", ], GYS: [ "Guangyuan", "广元", "GY|guangyuan|panlong", "GYS", "中国", "Guangyuan", "", "GYS", "0", "盘龙机场", "盘龙机场", ], TPE: [ "TaiBei", "台北(中国台湾)", "TB|taibei|中国|101大楼|阳明山公园|中正纪念堂|zhongguo|yangmingshangongyuan|zhongzhengjiniantang", "TPE", "中国台湾", "Taipei", "", "TPE", "0", "中国台湾台北桃园国际机场", "中国台湾台北桃园", ], NRT: [ "DongJing", "东京(成田)", "CT|DJ|Dongjing|Chengtian|Narita", "NRT", "日本", "Tokyo(Narita)", "", "NRT", "0", "东京成田国际机场", "东京成田", ], ENY: [ "Yanan", "延安", "YA|yanan|延安老区|南泥湾|延安", "ENY", "中国", "Yanan", "", "ENY", "0", "南泥湾机场", "延安南泥湾机场", ], RGN: [ "YangGuang", "仰光", "YG|Yangon|yangguang|miandian|缅甸", "RGN", "缅甸", "Yangon", "", "RGN", "0", "国际机场", "仰光", ], FUG: [ "Fuyang", "阜阳", "Fuyang", "FUG", "中国", "Fuyang", "", "FUG", "0", "西关机场", "阜阳西关机场", ], YYA: [ "YueYang", "岳阳", "YY", "YYA", "中国", "Yueyang", "", "YYA", "0", "三荷机场", "岳阳三荷机场", ], LYG: [ "LianYunGang", "连云港", "LYG", "LYG", "中国", "Lianyungang", "", "LYG", "0", "白塔埠机场", "连云港白塔埠机场", ], ZHY: [ "ZhongWei", "中卫", "ZW", "ZHY", "中国", "ZhongWei", "", "ZHY", "0", "沙坡头机场", "中卫沙坡头机场", ], WDS: [ "ShiYan", "十堰(武当山)", "sy|WDS|shiyan|wudangshan", "WDS", "中国", "Wudangshan", "", "WDS", "0", "十堰武当山机场", "十堰武当山机场", ], LYI: [ "LinYi", "临沂", "LY", "LYI", "中国", "Linyi", "", "LYI", "0", "启阳机场", "临沂启阳机场", ], BAV: [ "Baotou", "包头", "Baotou|BT|Donghe", "BAV", "中国", "Baotou", "", "BAV", "0", "东河机场", "包头东河机场", ], ZYI: [ "Zunyi", "遵义", "ZY|茅台|新舟|MT|XZ|maotai|XINZHOU", "ZYI", "中国", "Zunyi", "", "ZYI", "0", "新舟机场", "遵义新舟机场", ], WMT: [ "Zunyimaotai", "遵义茅台", "ZY|茅台|新舟|MT|XZ|maotai|XINZHOU", "WMT", "中国", "Zunyimaotai", "", "WMT", "0", "茅台机场", "遵义茅台机场", ], HKG: [ "XiangGang", "香港(中国香港)", "HK|xianggang|中国|海洋公园|太平山顶|维多利亚港|zhongguo|haiyanggongyuan|taipingshanding|weiduoliyagang", "HKG", "中国香港", "Hong Kong", "促销|购物|时尚", "HKG", "0", "香港国际机场", "香港", ], KMG: [ "KunMing", "昆明", "KM|中国|滇池|石林|zhongguo|dianchi|shilin", "KMG", "中国", "Kunming", "花市|大理|丽江", "KMG", "0", "长水国际机场", "昆明长水", ], YBP: [ "Yibin", "宜宾", "YB|中国|四川|宜宾|五粮液|酒都|zhongguo|sichuan|yibin|wuliangye|jiudu|", "YBP", "中国", "Yibin", "", "YBP", "0", "五粮液机场", "宜宾五粮液机场", ], CKG: [ "Chongqing", "重庆", "CQ|中国|洪崖洞|长江索道|解放碑|zhongguo|hongyadong|changjiangsuodao|jiefangbei", "CKG", "中国", "Chongqing", "解放碑|李子坝轻轨", "CKG", "0", "江北国际机场", "重庆江北", ], WEH: [ "WeiHai", "威海", "WH|Weihai|Dashuibo", "WEH", "中国", "Weihai", "", "WEH", "0", "大水泊机场", "威海大水泊机场", ], SQD: [ "Shangrao", "上饶", "SR|上饶|shangrao|三清山|sanqingshan", "SQD", "中国", "Shangrao", "", "SQD", "0", "三清山机场", "上饶三清山机场", ], KRY: [ "Kelamayi", "克拉玛依", "KLMY|Kelamayi|新疆|克拉玛依", "KRY", "中国", "Kelamayi", "", "KRY", "0", "机场", "克拉玛依机场", ], KHG: [ "Kashi", "喀什", "Kashi|KS|喀什|新疆|", "KHG", "中国", "Kashi", "", "KHG", "0", "机场", "喀什机场", ], SIA: [ "XiAn", "西安", "XA|中国|兵马俑|华清宫|鼓楼|zhongguo|bingmayong|huaqinggong|gulou", "SIA", "中国", "Xi'an", "", "SIA", "0", "咸阳国际机场", "西安咸阳", ], XIY: [ "XiAnJiChang", "西安机场", "XAJC", "XIY", "中国", "xianjichang", "", "XIY", "0", "咸阳国际机场", "西安咸阳", ], SHA: [ "ShangHai", "上海", "SH|中国|迪士尼|外滩|城隍庙|浦东|虹桥|pudong|hongqiao|zhongguo|dishini|waitan|chenghuangmiao", "SHA", "中国", "Shanghai", "魔都|金融|东方巴黎", "SHA", "0", "虹桥国际机场", "上海虹桥", ], PVG: [ "ShangHaiPuDong", "上海浦东", "SHPD", "PVG", "中国", "Shanghaipudong", "", "PVG", "0", "浦东国际机场", "上海浦东", ], OHE: [ "Mohe", "漠河", "Mohe|MH|漠河|黑龙江", "OHE", "中国", "Mohe", "", "OHE", "0", "机场", "漠河机场", ], }`
	t.Log(FormatJSONStr(s))
}

func TestCompareHideString(t *testing.T) {
	fmt.Println(CompareHideString("350204199806138023", "3502************23", "*"))
	fmt.Println(CompareHideString("3502************23", "350204199806138023", "*"))
	fmt.Println(CompareHideString("*****1", "11111*", "*"))
	fmt.Println(CompareHideString("你说什么", "你说**", "*"))
}

func TestPathJoin(t *testing.T) {
	tt := []struct {
		paths []string
		want  string
	}{
		{[]string{"a", "b", "c"}, "a/b/c"},
		{[]string{"a", "b", "c/"}, "a/b/c/"},
		{[]string{"http://www.example.com/", "/sub", "/item/"}, "http:/www.example.com/sub/item"},
	}

	for _, tc := range tt {
		if got := path.Join(tc.paths...); got != tc.want {
			t.Errorf("PathJoin(%v) = %v, want %v", tc.paths, got, tc.want)
		}
	}
}
