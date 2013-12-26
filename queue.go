package main

import (
	"errors"
	"fmt"
	"os"
	"path"
)

type Queue struct {
	basePath string // base path ...
}

func NewQueue(p string) (queue *Queue, err error) {
	// path p exists ?
	_, err = os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			// Queue does not exists try to create it
			if err = os.MkdirAll(p, os.ModeDir|0700); err != nil {
				return nil, errors.New(fmt.Sprintf("Path %s does not exist and i can't create it. %v", p, err))
			}
		} else {
			return nil, err
		}

	}

	// queue/msg
	msg := path.Join(p, "msg")
	if _, err = os.Stat(msg); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(msg, os.ModeDir|0700); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// queue/env
	env := path.Join(p, "env")
	if _, err = os.Stat(env); err != nil {
		if os.IsNotExist(err) {
			if err = os.Mkdir(env, os.ModeDir|0700); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &Queue{
		basePath: p,
	}, nil
}
