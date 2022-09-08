package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ostafen/demo/api"
	"github.com/ostafen/demo/model"
	"github.com/ostafen/demo/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type ClientConfig struct {
	Host string
}

func New(conf *ClientConfig) *TestClient {
	return &TestClient{
		conf: conf,
	}
}

type TestClient struct {
	conf *ClientConfig
}

func (c *TestClient) Create(answ *model.Answer) error {
	jsonBytes, err := json.Marshal(answ)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/answers", c.conf.Host), bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("answer already exist")
	}
	return nil
}

func (c *TestClient) Update(answ *model.Answer) error {
	jsonBytes, err := json.Marshal(answ)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s/answers/%s", c.conf.Host, answ.Key), "", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("answer does not exist")
	}
	return nil
}

func (c *TestClient) Get(key string) (*model.Answer, error) {
	resp, err := http.Get(fmt.Sprintf("%s/answers/%s", c.conf.Host, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no answer with key %s", key)
	}

	answ := &model.Answer{}
	err = json.NewDecoder(resp.Body).Decode(answ)
	return answ, err
}

func (c *TestClient) GetHistory(key string) ([]*model.Event, error) {
	resp, err := http.Get(fmt.Sprintf("%s/answers/%s/events", c.conf.Host, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []*model.Event
	err = json.NewDecoder(resp.Body).Decode(&events)
	return events, err
}

func (c *TestClient) Delete(key string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/answers/%s", c.conf.Host, key), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNoContent {
		return fmt.Errorf("no answer with key %s", key)
	}
	return nil
}

var clientConf = &ClientConfig{Host: "http://localhost:8080"}

func setupServer(t *testing.T) func() {
	dir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)

	s, err := store.Open(dir)
	require.NoError(t, err)

	controller := api.NewEventController(s)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	controller.Register(engine)

	hdr := engine.Handler()
	server := http.Server{Addr: ":8080", Handler: hdr}

	done := make(chan struct{}, 1)
	go func() {
		err := server.ListenAndServe()
		done <- struct{}{}

		if err == http.ErrServerClosed {
			err = nil
		}
		require.NoError(t, err)
	}()
	// add a small delay to ensure the background goroutine successfully calls ListenAndServe()
	// to prevent requests to be submitted to the server before it is started.
	time.Sleep(time.Millisecond * 10)

	return func() {
		require.NoError(t, server.Shutdown(context.Background()))
		<-done // ensure background goroutine successfully exited
		os.RemoveAll(dir)
	}
}

func TestCreateAndGetApi(t *testing.T) {
	done := setupServer(t)
	defer done()

	c := New(clientConf)

	answ := &model.Answer{Key: "myKey", Value: "myValue"}
	err := c.Create(answ)
	require.NoError(t, err)

	getAnsw, err := c.Get("myKey")
	require.NoError(t, err)
	require.Equal(t, getAnsw, answ)

	_, err = c.Get("myKey1")
	require.Error(t, err)

	// we cannot create an existing key
	err = c.Create(answ)
	require.Error(t, err)
}

func TestCreateUpdateAndDelete(t *testing.T) {
	done := setupServer(t)
	defer done()

	c := New(clientConf)

	answ := &model.Answer{Key: "myKey", Value: "myValue"}
	err := c.Create(answ)
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		updated := &model.Answer{Key: "myKey", Value: strconv.Itoa(i)}
		err := c.Update(updated)
		require.NoError(t, err)

		getAnsw, err := c.Get("myKey")
		require.NoError(t, err)
		require.Equal(t, getAnsw, updated)
	}

	err = c.Delete("myKey")
	require.NoError(t, err)

	// a non-existing key cannot be deleted
	err = c.Delete("myKey")
	require.Error(t, err)

	// get and updates should also fail for deleted keys
	_, err = c.Get("myKey")
	require.Error(t, err)

	err = c.Update(answ)
	require.Error(t, err)
}

func TestGetHistory(t *testing.T) {
	done := setupServer(t)
	defer done()

	c := New(clientConf)

	createAnsw := &model.Answer{Key: "myKey", Value: "initialValue"}
	err := c.Create(createAnsw)
	require.NoError(t, err)

	updateAnsw := &model.Answer{Key: "myKey", Value: "updatedValue"}
	err = c.Update(updateAnsw)
	require.NoError(t, err)

	// Get events are not recorded
	getAnsw, err := c.Get("myKey")
	require.NoError(t, err)
	require.Equal(t, getAnsw, updateAnsw)

	err = c.Delete(updateAnsw.Key)
	require.NoError(t, err)

	events, err := c.GetHistory("myKey")
	require.NoError(t, err)

	require.Len(t, events, 3)

	require.Equal(t, &model.Event{Event: model.CreateEvent, Data: createAnsw}, events[0])
	require.Equal(t, &model.Event{Event: model.UpdateEvent, Data: updateAnsw}, events[1])
	require.Equal(t, &model.Event{Event: model.DeleteEvent, Data: &model.Answer{Key: "myKey"}}, events[2])

	events, err = c.GetHistory("myKey1")
	require.NoError(t, err)

	require.Len(t, events, 0)
}
