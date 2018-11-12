package base

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/andres-erbsen/clock"
)

const (
	_testFileName = "test_file"
)

// Cleanup contains a list of function that are called to cleanup a fixture
type Cleanup struct {
	funcs []func()
}

// Add adds function to funcs list
func (c *Cleanup) Add(f ...func()) {
	c.funcs = append(c.funcs, f...)
}

// AppendFront append funcs from another cleanup in front of the funcs list
func (c *Cleanup) AppendFront(c1 *Cleanup) {
	c.funcs = append(c1.funcs, c.funcs...)
}

// Recover runs cleanup functions after test exit with exception
func (c *Cleanup) Recover() {
	if err := recover(); err != nil {
		c.run()
		panic(err)
	}
}

// Run runs cleanup functions when a test finishes running
func (c *Cleanup) Run() {
	c.run()
}

func (c *Cleanup) run() {
	for _, f := range c.funcs {
		f()
	}
}

func fileStatesFixture() (state1, state2, state3 FileState, run func()) {
	cleanup := &Cleanup{}
	defer cleanup.Recover()

	root, err := ioutil.TempDir("/tmp", "store_test")
	if err != nil {
		log.Fatal(err)
	}
	cleanup.Add(func() { os.RemoveAll(root) })

	state1Dir, err := ioutil.TempDir(root, "state1")
	if err != nil {
		log.Fatal(err)
	}

	state2Dir, err := ioutil.TempDir(root, "state2")
	if err != nil {
		log.Fatal(err)
	}

	state3Dir, err := ioutil.TempDir(root, "state3")
	if err != nil {
		log.Fatal(err)
	}

	state1 = NewFileState(state1Dir)
	state2 = NewFileState(state2Dir)
	state3 = NewFileState(state3Dir)

	return state1, state2, state3, cleanup.Run
}

type fileEntryTestBundle struct {
	state1 FileState
	state2 FileState
	state3 FileState

	entry FileEntry
}

func fileEntryLocalFixture() (bundle *fileEntryTestBundle, run func()) {
	cleanup := &Cleanup{}
	defer cleanup.Recover()

	state1, state2, state3, f := fileStatesFixture()
	cleanup.Add(f)
	entry := NewLocalFileEntryFactory().Create(_testFileName, state1)

	return &fileEntryTestBundle{
		state1: state1,
		state2: state2,
		state3: state3,
		entry:  entry,
	}, cleanup.Run
}

type fileMapTestBundle struct {
	state1 FileState
	state2 FileState
	state3 FileState

	entry FileEntry
	fm    FileMap
}

func fileMapSimpleFixture() (bundle *fileMapTestBundle, run func()) {
	cleanup := &Cleanup{}
	defer cleanup.Recover()

	b, clean := fileEntryLocalFixture()
	cleanup.Add(clean)

	fm := NewSimpleFileMap()

	return &fileMapTestBundle{
		state1: b.state1,
		state2: b.state2,
		state3: b.state3,
		entry:  b.entry,
		fm:     fm,
	}, cleanup.Run
}

func fileMapLRUFixture() (bundle *fileMapTestBundle, run func()) {
	cleanup := &Cleanup{}
	defer cleanup.Recover()

	b, clean := fileEntryLocalFixture()
	cleanup.Add(clean)

	fm := NewLRUFileMap(100, clock.New())

	return &fileMapTestBundle{
		state1: b.state1,
		state2: b.state2,
		state3: b.state3,
		entry:  b.entry,
		fm:     fm,
	}, cleanup.Run
}

// fileStoreBundle contains available states, FileStore and a map of FileEntry
// NOTE: do not use this struct directly, use fixtures instead
// TODO: breakdown fileStoreTestBundle
type fileStoreTestBundle struct {
	clk clock.Clock

	state1 FileState
	state2 FileState
	state3 FileState

	createStore func(clk clock.Clock) *localFileStore
	store       *localFileStore
	files       map[FileState]string
}

func (b *fileStoreTestBundle) recreateStore() {
	b.store = b.createStore(b.clk)
}

func fileStoreDefaultFixture() (*fileStoreTestBundle, func()) {
	return fileStoreFixture(func(clk clock.Clock) *localFileStore {
		store := NewLocalFileStore(clk)
		return store.(*localFileStore)
	})
}

func fileStoreCASFixture() (*fileStoreTestBundle, func()) {
	return fileStoreFixture(func(clk clock.Clock) *localFileStore {
		store := NewCASFileStore(clk)
		return store.(*localFileStore)
	})
}

func fileStoreLRUFixture(size int) (*fileStoreTestBundle, func()) {
	return fileStoreFixture(func(clk clock.Clock) *localFileStore {
		store := NewLRUFileStore(size, clk)
		return store.(*localFileStore)
	})
}

func fileStoreFixture(createStore func(clk clock.Clock) *localFileStore) (*fileStoreTestBundle, func()) {
	clk := clock.NewMock()
	store := createStore(clk)
	cleanup := &Cleanup{}
	defer cleanup.Recover()

	state1, state2, state3, f := fileStatesFixture()
	cleanup.Add(f)

	storeBundle := &fileStoreTestBundle{
		clk:         clk,
		state1:      state1,
		state2:      state2,
		state3:      state3,
		createStore: createStore,
		store:       store,
		files:       make(map[FileState]string),
	}

	// Create one test file in store
	err := storeBundle.store.NewFileOp().CreateFile(_testFileName, storeBundle.state1, 5)
	if err != nil {
		log.Fatal(err)
	}

	storeBundle.files[storeBundle.state1] = _testFileName

	return storeBundle, cleanup.Run
}
