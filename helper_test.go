package kuzu

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	once     sync.Once
	testDb   *Database
	testConn *Connection
)

var defaultNumThreads = uint64(4)
var floatEpsilon = 0.0000001

func initTinySNB(conn *Connection) error {
	tinySnbPath, err := filepath.Abs(filepath.Join("dataset", "tinysnb"))
	if err != nil {
		return err
	}
	// Normalize the path for Windows
	tinySnbPath = strings.ReplaceAll(tinySnbPath, "\\", "/")
	schemaPath := filepath.Join(tinySnbPath, "schema.cypher")
	err = executeCypherFromFile(schemaPath, conn, nil, nil)
	if err != nil {
		return err
	}

	copyPath := filepath.Join(tinySnbPath, "copy.cypher")
	originalPath := "dataset/tinysnb"
	err = executeCypherFromFile(copyPath, conn, &originalPath, &tinySnbPath)
	if err != nil {
		return err
	}

	conn.Query("create node table moviesSerial (ID SERIAL, name STRING, length INT32, note STRING, PRIMARY KEY (ID));")
	moviesSerialPath := filepath.Join(tinySnbPath, "vMoviesSerial.csv")
	// Normalize the path for Windows
	moviesSerialPath = strings.ReplaceAll(moviesSerialPath, "\\", "/")
	moviesSerialCopyQuery := fmt.Sprintf("copy moviesSerial from \"%s\"", moviesSerialPath)
	_, err = conn.Query(moviesSerialCopyQuery)
	if err != nil {
		return err
	}
	return nil
}

func executeCypherFromFile(filePath string, conn *Connection, originalString *string, replaceString *string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if originalString != nil && replaceString != nil {
			line = strings.ReplaceAll(line, *originalString, *replaceString)
		}
		_, err := conn.Query(line)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func SetupTestDatabase(t testing.TB) (*Database, *Connection) {
	t.Helper()
	once.Do(func() {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "testdb")
		// Normalize the path for Windows
		dbPath = strings.ReplaceAll(dbPath, "\\", "/")
		systemConfig := DefaultSystemConfig()
		systemConfig.BufferPoolSize = 256 * 1024 * 1024 // 256 MB
		systemConfig.MaxNumThreads = defaultNumThreads
		var err error
		testDb, err = OpenDatabase(dbPath, systemConfig)
		if err != nil {
			t.Fatalf("Error opening database: %v", err)
		}
		testConn, err = OpenConnection(testDb)
		if err != nil {
			t.Fatalf("Error opening connection: %v", err)
		}
		err = initTinySNB(testConn)
		if err != nil {
			t.Fatalf("Error initializing TinySNB: %v", err)
		}
	})
	return testDb, testConn
}
