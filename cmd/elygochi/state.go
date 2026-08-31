package main

import (
	"errors"
	"log"
	"os"
	"time"

	"elygochi/internal/pet"
	"elygochi/internal/statepath"
)

// sharedState wraps the pet save file so both the main window and the
// overlay can load their own in-memory copy and periodically persist it,
// picking up each other's changes on the next load/save cycle.
type sharedState struct {
	path string
}

func openSharedState() (*sharedState, pet.State) {
	path, err := statepath.Default()
	if err != nil {
		log.Fatalf("state: could not resolve save path: %v", err)
	}
	s := &sharedState{path: path}
	return s, s.load()
}

func (s *sharedState) load() pet.State {
	st, err := pet.Load(s.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("state: failed to load %s: %v", s.path, err)
		}
		st = pet.NewState(time.Now())
	}
	return st
}

func (s *sharedState) save(st pet.State) {
	if err := pet.Save(s.path, st); err != nil {
		log.Printf("state: failed to save %s: %v", s.path, err)
	}
}
