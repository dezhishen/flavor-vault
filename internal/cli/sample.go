package cli

import (
	"fmt"
	"time"

	"flavor-vault/internal/models"
	"flavor-vault/internal/store"
)

// writeSampleRecipe 写入示例菜谱（供 init 演示与前端测试）
func writeSampleRecipe(recipesDir string) error {
	ts := time.Now()
	samples := []*models.Recipe{
		{
			ID:          "hong-shao-rou",
			Name:        "红烧肉",
			Description: "经典家常红烧肉，肥而不腻、入口即化。",
			Tags:        []string{"热菜", "家常", "下饭菜", "宴客"},
			Kitchenware: []string{"炒锅", "砂锅"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "五花肉", Amount: "500g"},
					{Name: "冰糖", Amount: "30g"},
				},
				Side: []models.Ingredient{
					{Name: "生姜", Amount: "4片"},
					{Name: "八角", Amount: "2颗"},
					{Name: "生抽", Amount: "2勺"},
					{Name: "老抽", Amount: "1勺"},
					{Name: "料酒", Amount: "2勺"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "五花肉切块，冷水下锅焯水，捞出洗净。"},
				{Order: 2, Description: "炒锅放少许油，下冰糖小火炒出糖色。"},
				{Order: 3, Description: "下五花肉翻炒上色，加入姜片、八角、料酒、生抽、老抽。"},
				{Order: 4, Description: "转入砂锅，加热水没过肉块，小火炖 1 小时。"},
				{Order: 5, Description: "大火收汁，出锅装盘。"},
			},
			Media: models.Media{Cover: "images/hong-shao-rou.jpg"},
			Stats: models.Stats{PrepTime: 20, CookTime: 70, Difficulty: 3},
		},
		{
			ID:          "pai-huang-gua",
			Name:        "拍黄瓜",
			Description: "清爽开胃的夏日凉菜，五分钟搞定。",
			Tags:        []string{"凉菜", "快手", "家常"},
			Kitchenware: []string{"保鲜袋"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "黄瓜", Amount: "2根"},
					{Name: "大蒜", Amount: "4瓣"},
				},
				Side: []models.Ingredient{
					{Name: "生抽", Amount: "2勺"},
					{Name: "香醋", Amount: "1勺"},
					{Name: "香油", Amount: "1小勺"},
					{Name: "辣椒油", Amount: "适量"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "黄瓜洗净，用刀背拍裂后切段。"},
				{Order: 2, Description: "大蒜拍碎切末。"},
				{Order: 3, Description: "加入生抽、香醋、香油、辣椒油拌匀，冷藏 10 分钟更佳。"},
			},
			Media: models.Media{Cover: "images/pai-huang-gua.jpg"},
			Stats: models.Stats{PrepTime: 10, CookTime: 0, Difficulty: 1},
		},
		{
			ID:          "fan-qie-chao-dan",
			Name:        "番茄炒蛋",
			Description: "国民家常菜，酸甜可口，超级下饭。",
			Tags:        []string{"热菜", "家常", "快手"},
			Kitchenware: []string{"炒锅"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "番茄", Amount: "3个"},
					{Name: "鸡蛋", Amount: "3个"},
				},
				Side: []models.Ingredient{
					{Name: "小葱", Amount: "1根"},
					{Name: "白糖", Amount: "1小勺"},
					{Name: "盐", Amount: "适量"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "番茄切块，鸡蛋打散加少许盐。"},
				{Order: 2, Description: "热锅冷油，倒入蛋液炒至凝固盛出。"},
				{Order: 3, Description: "锅内下番茄翻炒出汁，加糖和盐调味。"},
				{Order: 4, Description: "倒入鸡蛋翻炒均匀，撒葱花出锅。"},
			},
			Media: models.Media{Cover: "images/fan-qie-chao-dan.jpg"},
			Stats: models.Stats{PrepTime: 10, CookTime: 10, Difficulty: 1},
		},
		{
			ID:          "ma-po-dou-fu",
			Name:        "麻婆豆腐",
			Description: "麻辣鲜香的川味名菜，豆腐嫩滑入味。",
			Tags:        []string{"热菜", "川菜", "下饭菜"},
			Kitchenware: []string{"炒锅"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "嫩豆腐", Amount: "400g"},
					{Name: "牛肉末", Amount: "100g"},
				},
				Side: []models.Ingredient{
					{Name: "豆瓣酱", Amount: "1勺"},
					{Name: "花椒粉", Amount: "1小勺"},
					{Name: "蒜苗", Amount: "2根"},
					{Name: "淀粉", Amount: "适量"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "豆腐切块，盐水焯烫后沥干。"},
				{Order: 2, Description: "炒锅下油，煸炒牛肉末至酥香。"},
				{Order: 3, Description: "下豆瓣酱炒出红油，加适量水烧开。"},
				{Order: 4, Description: "下豆腐小火煮 3 分钟，勾芡撒花椒粉和蒜苗。"},
			},
			Media: models.Media{Cover: "images/ma-po-dou-fu.jpg"},
			Stats: models.Stats{PrepTime: 10, CookTime: 15, Difficulty: 2},
		},
		{
			ID:          "kong-qi-zha-ji-chi",
			Name:        "空气炸锅鸡翅",
			Description: "不用一滴油，外酥里嫩的烤鸡翅。",
			Tags:        []string{"热菜", "快手", "家常"},
			Kitchenware: []string{"空气炸锅"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "鸡翅中", Amount: "8个"},
				},
				Side: []models.Ingredient{
					{Name: "生抽", Amount: "2勺"},
					{Name: "蚝油", Amount: "1勺"},
					{Name: "蜂蜜", Amount: "1勺"},
					{Name: "黑胡椒", Amount: "适量"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "鸡翅两面划刀，加入调料腌制 30 分钟。"},
				{Order: 2, Description: "空气炸锅 200℃ 预热 3 分钟。"},
				{Order: 3, Description: "放入鸡翅，200℃ 烤 15 分钟，中途翻面。"},
				{Order: 4, Description: "出炉撒黑胡椒即可。"},
			},
			Media: models.Media{Cover: "images/kong-qi-zha-ji-chi.jpg"},
			Stats: models.Stats{PrepTime: 35, CookTime: 15, Difficulty: 1},
		},
		{
			ID:          "lao-huo-tang",
			Name:        "玉米胡萝卜排骨汤",
			Description: "清甜滋润的老火汤，营养又暖胃。",
			Tags:        []string{"汤羹", "家常", "宴客"},
			Kitchenware: []string{"砂锅"},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{
					{Name: "排骨", Amount: "500g"},
					{Name: "甜玉米", Amount: "2根"},
					{Name: "胡萝卜", Amount: "1根"},
				},
				Side: []models.Ingredient{
					{Name: "生姜", Amount: "3片"},
					{Name: "盐", Amount: "适量"},
				},
			},
			Steps: []models.Step{
				{Order: 1, Description: "排骨冷水下锅焯水，捞出洗净。"},
				{Order: 2, Description: "玉米、胡萝卜切段。"},
				{Order: 3, Description: "所有材料放入砂锅，加足量水大火烧开。"},
				{Order: 4, Description: "转小火炖 1.5 小时，加盐调味。"},
			},
			Media: models.Media{Cover: "images/lao-huo-tang.jpg"},
			Stats: models.Stats{PrepTime: 15, CookTime: 90, Difficulty: 2},
		},
	}

	fs := store.NewRecipeFileStore(recipesDir)
	for _, r := range samples {
		r.CreatedAt = ts
		r.UpdatedAt = ts
		if err := fs.Save(r); err != nil {
			return fmt.Errorf("写入示例菜谱 %s 失败: %w", r.ID, err)
		}
	}
	return nil
}

