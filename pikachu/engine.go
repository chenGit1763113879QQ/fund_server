package pikachu

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/vmihailenco/msgpack"
)

type Engine struct {
	colls []*collection
}

var engine *Engine

func init() {
	engine = &Engine{colls: make([]*collection, 0, MAX_COLLECTION_NUM)}

	engine.loadFromFile()
	go func() {
		time.Sleep(STORE_DURATION)
		engine.saveAsFile()
	}()
}

func Coll(name string) *collection {
	for _, c := range engine.colls {
		if name == c.Name {
			return c
		}
	}

	// if not exist, create
	c, err := CreateColl(name)
	if err != nil {
		return nil
	}
	return c
}

func CreateColl(name string) (*collection, error) {
	if len(engine.colls) == MAX_COLLECTION_NUM {
		return nil, errors.New("too many collections")
	}

	for _, c := range engine.colls {
		if name == c.Name {
			return nil, errors.New("reqiured unique collection name")
		}
	}

	c := &collection{Name: name, Data: make(map[string]any)}
	engine.colls = append(engine.colls, c)

	return c, nil
}

func (e *Engine) saveAsFile() {
	res, _ := msgpack.Marshal(e.colls)
	if err := ioutil.WriteFile(STORE_FILE_NAME, res, os.ModePerm); err != nil {
		panic(err)
	}
}

func (e *Engine) loadFromFile() {
	file, err := os.ReadFile(STORE_FILE_NAME)
	if err != nil {
		return
	}
	err = msgpack.Unmarshal(file, &e.colls)
	if err != nil {
		panic(err)
	}
}
