package filesystem

import (
	"fmt"
	"sync"

	"github.com/owles/go-storage/contract"
)

type Storage struct {
	filesystem.Driver
	defaultDisk string
	timezone    string
	drivers     map[string]filesystem.Driver
}

var storage *Storage
var mu sync.Mutex

func FS() filesystem.Storage {
	mu.Lock()
	defer mu.Unlock()

	if storage == nil {
		storage = &Storage{
			drivers: make(map[string]filesystem.Driver),
		}
	}

	return storage
}

func (s *Storage) AddDriver(disk string, driver filesystem.Driver) filesystem.Storage {
	mu.Lock()
	defer mu.Unlock()

	s.drivers[disk] = driver

	return s
}

func (s *Storage) SetDefaultDisk(disk string) filesystem.Storage {
	mu.Lock()
	defer func() {
		mu.Unlock()
		s.Driver = s.Disk(disk)
	}()

	s.defaultDisk = disk

	return s
}

func (s *Storage) SetTimezone(timezone string) filesystem.Storage {
	mu.Lock()
	defer mu.Unlock()

	s.timezone = timezone

	return s
}

func (s *Storage) Disk(disk string) filesystem.Driver {
	mu.Lock()
	defer mu.Unlock()

	if driver, exist := s.drivers[disk]; exist {
		return driver
	}

	panic(fmt.Errorf("disk doesn't exist"))
}
