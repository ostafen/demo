package api

import (
	"bufio"
	"encoding/json"
	"net/http"

	"github.com/ostafen/demo/model"
	"github.com/ostafen/demo/store"

	"github.com/gin-gonic/gin"
)

type EventController struct {
	store store.EventStore
}

func NewEventController(store store.EventStore) *EventController {
	return &EventController{
		store: store,
	}
}

func (c *EventController) CreateAnswer(ctx *gin.Context) {
	var answ model.Answer

	if err := ctx.BindJSON(&answ); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := c.store.Create(&answ); err != nil {
		if err == store.ErrAnswerExist {
			ctx.AbortWithError(http.StatusConflict, err)
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}
	ctx.IndentedJSON(http.StatusCreated, answ)
}

func (c *EventController) DeleteAnswer(ctx *gin.Context) {
	key := ctx.Param("key")

	if err := c.store.Delete(key); err != nil {
		if err == store.ErrAnswerNotExist {
			ctx.AbortWithError(http.StatusNoContent, err)
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}
}

func (c *EventController) GetHistory(ctx *gin.Context) {
	key := ctx.Param("key")

	writer := bufio.NewWriter(ctx.Writer)

	it, err := c.store.GetHistory(key)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.Header("Content-Type", "application/json")
	if err := c.writeEvents(writer, it); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (c *EventController) writeEvents(writer *bufio.Writer, it store.EventIterator) error {
	if _, err := writer.WriteString("["); err != nil {
		return err
	}

	hasData := false
	for it.Next() {
		if hasData {
			writer.WriteString(",")
		} else {
			hasData = true
		}

		e, err := it.Value()
		if err != nil {
			return err
		}

		evtJson, err := json.Marshal(e)
		if err != nil {
			return err
		}

		if _, err := writer.WriteString(string(evtJson)); err != nil {
			return err
		}
	}

	_, err := writer.WriteString("]")
	if err != nil {
		return err
	}
	return writer.Flush()
}

func (c *EventController) GetAnswer(ctx *gin.Context) {
	key := ctx.Param("key")

	answ, err := c.store.GetAnswer(key)
	if err != nil {
		if err == store.ErrAnswerNotExist {
			ctx.AbortWithError(http.StatusNotFound, err)
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, answ)
}

func (c *EventController) UpdateAnswer(ctx *gin.Context) {
	var answ model.Answer

	if err := ctx.BindJSON(&answ); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err := c.store.Update(&answ)
	if err != nil {
		if err == store.ErrAnswerNotExist {
			ctx.AbortWithError(http.StatusNotFound, err)
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, answ)
}

func (c *EventController) Register(engine *gin.Engine) {
	engine.PUT("/answers", c.CreateAnswer)
	engine.GET("/answers/:key", c.GetAnswer)
	engine.POST("/answers/:key", c.UpdateAnswer)
	engine.DELETE("/answers/:key", c.DeleteAnswer)
	engine.GET("/answers/:key/events", c.GetHistory)
}
