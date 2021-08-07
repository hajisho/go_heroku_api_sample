package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"

	//データベース
	"github.com/jinzhu/gorm"
)

var count int

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	//db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	//if err != nil {
	//	log.Fatalf("Error opening database: %q", err)
	//}
	dbInit() //データベースマイグレート
	AddRecodeInit()

	router := gin.New()
	router.Use(gin.Logger())
	//router.LoadHTMLGlob("templates/*.tmpl.html")
	//router.Static("/static", "static")

	//ルートは404を返す
	//router.GET("/", returnHello)
	r1 := router.Group("/:tmp_slash")
	r := r1.Group("/recipes")
	{
		//レシピを作成
		r.POST("", createRecipe)
		//全てのレシピ一覧を返す
		r.GET("", returnRecipeALL)
		//指定のレシピ一つを返す
		r.GET("/:id", returnRecipe)
		//指定のレシピを更新
		r.PATCH("/:id", updateRecipe)
		//指定のレシピ/を削除
		r.DELETE("/:id", deleteRecipe)
	}

	router.Run(":" + port)
}

type RecipeInfo struct {
	Rid         int    `json:"id"`
	Title       string `json:"title"`
	Making_time string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        string `json:"cost"`
}

type NewRecipeInfo struct {
	Title       string `json:"title"`
	Making_time string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        int    `json:"cost"`
}

func AddRecodeInit() {
	//
	registRecipe(1, "メニュー1", "5分", "4人", "卵、ベーコン", 100, "2000-05-11 11:19:14", "2000-05-11 11:19:14")
	registRecipe(2, "メニュー2", "10分", "1人", "玉ねぎ,卵,醤油", 70, "2001-01-01 03:11:16", "2001-01-01 03:11:16")
	count = 2
}

type PostErrorJson struct {
	Message  string `json:"message"`
	Required string `json:"required"`
}
type PostJson struct {
	Message string              `json:"message"`
	Recipe  []CreatedRecipeInfo `json:"recipe"`
}

type CreatedRecipeInfo struct {
	Rid         int    `json:"id"`
	Title       string `json:"title"`
	Making_time string `json:"making_time"`
	Serves      string `json:"serves"`
	Ingredients string `json:"ingredients"`
	Cost        string `json:"cost"`
	Create      string `json:"created_at"`
	Update      string `json:"updated_at"`
}

var layout = "2016-01-12 14:10:12"

func timeToString(t time.Time) string {
	str := t.Format(layout)
	return str
}

func createRecipe(c *gin.Context) {
	// POST bodyからメッセージを獲得
	req := new(NewRecipeInfo)
	err := c.BindJSON(req)
	if err != nil {
		// メッセージがJSONではない、もしくは、content-typeがapplication/jsonになっていない
		fmt.Println("stop at createRecipe")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed request as JSON format is expected"})
		return
	}

	title := req.Title
	making_time := req.Making_time
	serves := req.Serves
	ingredients := req.Ingredients
	cost := req.Cost
	create := timeToString(time.Now())
	update := timeToString(time.Now())

	if title == "" || making_time == "" || serves == "" || ingredients == "" || cost == 0 {

		posterrorjson := PostErrorJson{
			Message:  "Recipe creation failed!",
			Required: "title, making_time, serves, ingredients, cost",
		}
		// メッセージがない、無効なリクエスト
		c.JSON(http.StatusOK, posterrorjson)
		return
	}
	count += 1
	registRecipe(count, title, making_time, serves, ingredients, cost, create, update)
	recipe := make([]CreatedRecipeInfo, 1)
	recipe[0] = CreatedRecipeInfo{
		//Rid:         int(dbGetOneByTitle(title).Rid),
		Rid:         count,
		Title:       title,
		Making_time: making_time,
		Serves:      serves,
		Ingredients: ingredients,
		Cost:        strconv.Itoa(cost),
		Create:      create,
		Update:      update,
	}
	postjson := PostJson{
		Message: "Recipe successfully created!",
		Recipe:  recipe,
	}
	c.JSON(http.StatusOK, postjson)
	return
}

type AllRecipeJson struct {
	Recipe []RecipeInfo `json:"recipes"`
}

func returnRecipeALL(c *gin.Context) {
	recipeInDB := dbGetAll()
	recipes := make([]RecipeInfo, len(recipeInDB))

	for i, rcp := range recipeInDB {
		recipes[i] = RecipeInfo{
			Rid:         int(rcp.Rid),
			Title:       rcp.Title,
			Making_time: rcp.MakingTime,
			Serves:      rcp.Serves,
			Ingredients: rcp.Ingredients,
			Cost:        strconv.Itoa(rcp.Cost),
		}
	}
	recipesJson := AllRecipeJson{
		Recipe: recipes,
	}
	c.JSON(http.StatusOK, recipesJson)
}

type ARecipeJson struct {
	Message string       `json:"message"`
	Recip   []RecipeInfo `json:"recipe"`
}

func returnRecipe(c *gin.Context) {
	string_id := c.Param("id")
	id, _ := strconv.Atoi(string_id)

	rcp := dbGetOne(id)
	// データベースに保存されているメッセージの形式から、クライアントへ返す形式に変換する
	recipe := make([]RecipeInfo, 1)
	recipe[0] = RecipeInfo{
		Rid:         int(rcp.Rid),
		Title:       rcp.Title,
		Making_time: rcp.MakingTime,
		Serves:      rcp.Serves,
		Ingredients: rcp.Ingredients,
		Cost:        strconv.Itoa(rcp.Cost),
	}

	recipeJson := ARecipeJson{
		Message: "Recipe details by id",
		Recip:   recipe,
	}

	c.JSON(http.StatusOK, recipeJson)
}

type PatchJson struct {
	Message string       `json:"message"`
	Recipe  []RecipeInfo `json:"recipe"`
}

func updateRecipe(c *gin.Context) {
	string_id := c.Param("id")
	id, _ := strconv.Atoi(string_id)

	// POST bodyからメッセージを獲得
	req := new(NewRecipeInfo)
	err := c.BindJSON(req)
	if err != nil {
		// メッセージがJSONではない、もしくは、content-typeがapplication/jsonになっていない
		fmt.Println("stop at createRecipe")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Malformed request as JSON format is expected"})
		return
	}

	title := req.Title
	making_time := req.Making_time
	serves := req.Serves
	ingredients := req.Ingredients
	cost := req.Cost
	update := timeToString(time.Now())

	//title := c.Param("title")
	//making_time := c.Param("making_time")
	//serves := c.Param(("serves"))
	//ingredients := c.Param("ingredients")
	//cost := c.Param(("cost"))

	if title == "" || making_time == "" || serves == "" || ingredients == "" || cost == 0 {

		posterrorjson := PostErrorJson{
			Message:  "Recipe creation failed!",
			Required: "title, making_time, serves, ingredients, cost",
		}
		// メッセージがない、無効なリクエスト
		c.JSON(http.StatusOK, posterrorjson)
		return
	}

	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	recipe := Recipe{}
	db.Model(&recipe).Where("rid = ?", id).Update("title", title)
	db.Model(&recipe).Where("rid = ?", id).Update("making_time", making_time)
	db.Model(&recipe).Where("rid = ?", id).Update("serves", serves)
	db.Model(&recipe).Where("rid = ?", id).Update("ingredients", ingredients)
	db.Model(&recipe).Where("rid = ?", id).Update("cost", cost)
	db.Model(&recipe).Where("rid = ?", id).Update("updatetime", update)

	recipe2 := make([]RecipeInfo, 1)
	recipe2[0] = RecipeInfo{
		Rid:         id,
		Title:       title,
		Making_time: making_time,
		Serves:      serves,
		Ingredients: ingredients,
		Cost:        strconv.Itoa(cost),
	}

	patchjson := PatchJson{
		Message: "Recipe successfully updated!",
		Recipe:  recipe2,
	}
	c.JSON(http.StatusOK, patchjson)

}

func deleteRecipe(c *gin.Context) {
	string_id := c.Param("id")
	id, _ := strconv.Atoi(string_id)
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	recipe := getRecipeByID(id)
	if recipe.Rid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not exist"})
		return
	}
	db.Delete(&recipe)
	db.Close()
	//0を削除とする=> GETでrid=0は非表示
	//if err := db.Model(&recipe).Where("rid = ?", id).Update("rid", 0).Error; err != nil {
	//	c.JSON(http.StatusOK, gin.H{"message": "No Recipe found"})
	//	return
	//}
	c.JSON(http.StatusOK, gin.H{"message": "Recipe successfully removed!"})
	return

}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////
type Recipe struct {
	gorm.Model
	Rid         int    `gorm:"not null"`
	Title       string `gorm:"not null"`
	MakingTime  string `gorm:"not null"`
	Serves      string `gorm:"not null"`
	Ingredients string `gorm:"not null"`
	Cost        int    `gorm:"not null"`
	CreateTime  string `gorm:"not null"`
	UpdateTime  string `gorm:"not null"`
}

//DBマイグレート
//main関数の最初でdbInit()を呼ぶことでデータベースマイグレート
func dbInit() {
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		panic("データベース開ません(dbinit)")
	}
	db.AutoMigrate(&Recipe{})
	defer db.Close()
}

// レシピ登録処理
func registRecipe(rid int, title string, making_time string, serves string, ingredients string, cost int, create string, update string) error {
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Insert処理
	fmt.Println("------------")
	fmt.Println(count)
	if err := db.Create(&Recipe{Rid: rid, Title: title, MakingTime: making_time, Serves: serves, Ingredients: ingredients, Cost: cost, CreateTime: create, UpdateTime: update}).Error; err != nil {
		return err
	}
	return nil
}

//レシピを全取得
func dbGetAll() []Recipe {
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var recipes []Recipe
	//カラム名は自動でスネークケース
	db.Not("rid = ?", 0).Order("rid").Find(&recipes)
	return recipes
}

//指定したIDのレシピを返す
func dbGetOne(id int) Recipe {
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	var recipe Recipe
	db.Where("rid = ?", id).First(&recipe)
	db.Close()
	return recipe
}

//指定したtitleのレシピを返す
//func dbGetOneByTitle(title string) Recipe {
//	databaseUrl := os.Getenv("DATABASE_URL")
//	db, err := gorm.Open("postgres", databaseUrl)
//	if err != nil {
//		log.Fatal(err)
//	}
//	var recipe Recipe
//	db.Where("title = ?", title).First(&recipe)
//	db.Close()
//	return recipe
//}

// getUserById は、指定されたIDを持つユーザーを一つ返します。
// ユーザーが存在しない場合、IDが0のレコードが返ります。
func getRecipeByID(id int) Recipe {
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err := gorm.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var recipe Recipe
	db.Where("rid = ?", id).First(&recipe)
	return recipe
}
