package storage

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ContentWatcher watches a directory for changes and triggers sync callbacks
type ContentWatcher struct {
	watcher    *fsnotify.Watcher
	contentDir string
	callback   func() error
	debounce   time.Duration

	// Debounce state
	mu          sync.Mutex
	timer       *time.Timer
	pendingSync bool

	done chan struct{}
	wg   sync.WaitGroup
}

// NewContentWatcher creates a new file system watcher
func NewContentWatcher(contentDir string, callback func() error) (*ContentWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	cw := &ContentWatcher{
		watcher:    watcher,
		contentDir: contentDir,
		callback:   callback,
		debounce:   500 * time.Millisecond, // Debounce rapid changes
		done:       make(chan struct{}),
	}

	return cw, nil
}

// Start begins watching for file changes
func (cw *ContentWatcher) Start() error {
	// Add the content directory to the watcher
	if err := cw.watcher.Add(cw.contentDir); err != nil {
		return err
	}

	cw.wg.Add(1)
	go cw.watch()

	log.Printf("File watcher started for: %s", cw.contentDir)
	return nil
}

// Stop stops the watcher
func (cw *ContentWatcher) Stop() error {
	close(cw.done)
	cw.wg.Wait()

	cw.mu.Lock()
	if cw.timer != nil {
		cw.timer.Stop()
	}
	cw.mu.Unlock()

	return cw.watcher.Close()
}

// watch runs the main event loop
func (cw *ContentWatcher) watch() {
	defer cw.wg.Done()

	for {
		select {
		case <-cw.done:
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			cw.handleEvent(event)

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// handleEvent processes a filesystem event
func (cw *ContentWatcher) handleEvent(event fsnotify.Event) {
	// Only care about markdown files
	if !strings.HasSuffix(event.Name, ".md") {
		return
	}

	// Ignore temporary/backup files
	base := filepath.Base(event.Name)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return
	}

	// Only sync on write, create, remove, or rename
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}

	log.Printf("File changed: %s (%s)", event.Name, event.Op)
	cw.triggerSync()
}

// triggerSync schedules a sync with debouncing
func (cw *ContentWatcher) triggerSync() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Reset the debounce timer
	if cw.timer != nil {
		cw.timer.Stop()
	}

	cw.timer = time.AfterFunc(cw.debounce, func() {
		cw.mu.Lock()
		cw.pendingSync = false
		cw.mu.Unlock()

		if err := cw.callback(); err != nil {
			log.Printf("Sync error: %v", err)
		} else {
			log.Println("Content synced after file change")
		}
	})
	cw.pendingSync = true
}
