package main

import (
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	_ "github.com/vtotbl/mini-backend/docs"
)

type ResponseID struct {
	ID int64 `json:"id"`
}

var sequence atomic.Int64

var cacheMemory = cache.New(time.Hour, time.Hour)

func main() {
	router := gin.Default()

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, Geo!",
		})
	})

	router.POST("/add", Add)
	router.POST("/add-in-table/:table", AddInTable)
	router.GET("/get-all", GetAll)
	router.GET("/get-by-id/:id", GetByID)
	router.GET("/get-by-table/:table", GetByTable)
	router.GET("get-by-table-and-id/:table/:id", GetByTableAndID)

	router.Run(":8080")
}

// Add
// @Summary Add
// @Tags basic
// @Description Добавление записи
// @ID Add
// @Param input body any true "Объект для записи"
// @Accept json
// @Produce json
// @Success 200 {object} ResponseID
// @Failure 400 {object} error
// @Failure 500 {object} error
// @Router /add [post]
func Add(c *gin.Context) {
	var data any
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, errors.Wrap(err, "bind json"))
	}

	id := sequence.Add(1)

	if err := cacheMemory.Add(strconv.Itoa(int(id)), data, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, errors.Wrap(err, "cache add"))
	}

	c.JSON(http.StatusOK, ResponseID{ID: id})
}

// AddInTable
// @Summary AddInTable
// @Tags by table
// @Description Добавление записи в подхранилище
// @ID AddInTable
// @Accept json
// @Produce json
// @Param input body any true "Объект для записи"
// @Param table path string true "Table"
// @Success 200 {object} ResponseID
// @Failure 400 {object} error
// @Failure 500 {object} error
// @Router /add-in-table/{table} [post]
func AddInTable(c *gin.Context) {
	tableName := c.Param("table")

	var data any
	if err := c.ShouldBindJSON(&data); err != nil {
		println(err)
		c.JSON(http.StatusBadRequest, errors.Wrap(err, "bind json"))
		return
	}

	ifNoAdd(c, tableName)

	if err := cacheMemory.Increment(tableName, 1); err != nil {
		c.JSON(http.StatusInternalServerError, errors.Wrap(err, "Increment"))
		return
	}

	id, ok := cacheMemory.Get(tableName)
	if !ok {
		c.JSON(http.StatusInternalServerError, "get error")
		return
	}

	if err := cacheMemory.Add(tableName+"_"+strconv.Itoa(int(id.(int64))), data, 10*time.Minute); err != nil {
		c.JSON(http.StatusInternalServerError, errors.Wrap(err, "cache add"))
		return
	}

	c.JSON(http.StatusOK, ResponseID{ID: id.(int64)})
}

func ifNoAdd(c *gin.Context, table string) {
	_, ok := cacheMemory.Get(table)
	if !ok {
		if err := cacheMemory.Add(table, int64(0), time.Hour); err != nil {
			c.JSON(http.StatusInternalServerError, err)
		}
	}
}

// GetAll
// @Summary GetAll
// @Tags basic
// @Description Получение всех объектов
// @ID GetAll
// @Accept json
// @Produce json
// @Success 200 {object} []any
// @Router /get-all [get]
func GetAll(c *gin.Context) {
	objs := make([]any, 0)

	i := 1
	for {
		key := strconv.Itoa(i)
		val, ok := cacheMemory.Get(key)
		if !ok {
			break
		}

		objs = append(objs, val)
		i++
	}

	c.JSON(http.StatusOK, objs)
}

// GetByID
// @Summary GetByID
// @Tags basic
// @Description Получение всех объектов
// @ID GetByID
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} any
// @Router /get-by-id/{id} [get]
func GetByID(c *gin.Context) {
	id := c.Param("id")

	val, ok := cacheMemory.Get(id)
	if !ok {
		c.JSON(http.StatusInternalServerError, "error get from memory")
	}

	c.JSON(http.StatusOK, val)
}

// GetByTable
// @Summary GetByTable
// @Tags by table
// @Description Получение всех объектов таблицы
// @ID GetByTable
// @Accept json
// @Produce json
// @Param table path string true "table"
// @Success 200 {object} any
// @Router /get-by-table/{table} [get]
func GetByTable(c *gin.Context) {
	table := c.Param("table")

	objs := make([]any, 0)

	i := 1
	for {
		key := strconv.Itoa(i)
		val, ok := cacheMemory.Get(table + "_" + key)
		if !ok {
			break
		}

		objs = append(objs, val)
		i++
	}

	c.JSON(http.StatusOK, objs)
}

// GetByTableAndID
// @Summary GetByTableAndID
// @Tags by table
// @Description Получение всех объектов таблицы
// @ID GetByTableAndID
// @Accept json
// @Produce json
// @Param table path string true "table"
// @Param id path int true "id"
// @Success 200 {object} any
// @Router /get-by-table-and-id/{table}/{id} [get]
func GetByTableAndID(c *gin.Context) {
	table := c.Param("table")

	id := c.Param("id")

	val, ok := cacheMemory.Get(table + "_" + id)
	if !ok {
		c.JSON(http.StatusInternalServerError, "error get from memory")
	}

	c.JSON(http.StatusOK, val)
}
