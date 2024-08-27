package storage

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx"
	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"github.com/vkupriya/go-metrics/internal/server/models"
)

func TestMain(m *testing.M) {
	code, err := runMain(m)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(code)
}

const (
	testDBName       = "test"
	testUserName     = "test"
	testUserPassword = "test"
)

var (
	getDSN          func() string
	getSUConnection func() (*pgx.Conn, error)
)

func initGetDSN(hostAndPort string) {
	getDSN = func() string {
		return fmt.Sprintf(
			"postgres://%s:%s@%s/%s?sslmode=disable",
			testUserName,
			testUserPassword,
			hostAndPort,
			testDBName,
		)
	}
}

func initGetSUConnection(hostPort string) error {
	host, port, err := getHostPort(hostPort)
	if err != nil {
		return fmt.Errorf("failed to extract the host and port parts from the string %s: %w", hostPort, err)
	}
	getSUConnection = func() (*pgx.Conn, error) {
		conn, err := pgx.Connect(pgx.ConnConfig{
			Host:     host,
			Port:     port,
			Database: "postgres",
			User:     "postgres",
			Password: "postgres",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get a super user connection: %w", err)
		}
		return conn, nil
	}
	return nil
}

func runMain(m *testing.M) (int, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return 1, fmt.Errorf("failed to initialize a pool: %w", err)
	}

	pg, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "16",
			Name:       "migration-integration-tests",
			Env: []string{
				"POSTGRES_USER=postgres",
				"POSTGRES_PASSWORD=postgres",
			},
			ExposedPorts: []string{"5432/tcp"},
		},
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return 1, fmt.Errorf("failed to run the postgres container: %w", err)
	}

	defer func() {
		if err := pool.Purge(pg); err != nil {
			log.Printf("failed to purge the postgres container: %v", err)
		}
	}()

	hostPort := pg.GetHostPort("5432/tcp")
	initGetDSN(hostPort)
	if err := initGetSUConnection(hostPort); err != nil {
		return 1, fmt.Errorf("failed to connect as admin to postgresql DB: %w", err)
	}

	pool.MaxWait = 10 * time.Second
	var conn *pgx.Conn
	if err := pool.Retry(func() error {
		conn, err = getSUConnection()
		if err != nil {
			return fmt.Errorf("failed to connect to the DB: %w", err)
		}
		return nil
	}); err != nil {
		return 1, fmt.Errorf("failed to connect to the DB: %w", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to correctly close the connection: %v", err)
		}
	}()

	if err := createTestDB(conn); err != nil {
		return 1, fmt.Errorf("failed to create a test DB: %w", err)
	}

	exitCode := m.Run()

	return exitCode, nil
}

func createTestDB(conn *pgx.Conn) error {
	_, err := conn.Exec(
		fmt.Sprintf(
			`CREATE USER %s PASSWORD '%s'`,
			testUserName,
			testUserPassword,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create a test user: %w", err)
	}

	_, err = conn.Exec(
		fmt.Sprintf(`
			CREATE DATABASE %s
				OWNER '%s'
				ENCODING 'UTF8'
				LC_COLLATE = 'en_US.utf8'
				LC_CTYPE = 'en_US.utf8'
			`, testDBName, testUserName,
		),
	)

	if err != nil {
		return fmt.Errorf("failed to create a test DB: %w", err)
	}

	return nil
}

func getHostPort(hostPort string) (string, uint16, error) {
	hostPortParts := strings.Split(hostPort, ":")
	if len(hostPortParts) != 2 {
		return "", 0, fmt.Errorf("got an invalid host-port string: %s", hostPort)
	}

	portStr := hostPortParts[1]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to cast the port %s to an int: %w", portStr, err)
	}
	return hostPortParts[0], uint16(port), nil
}

//nolint:dupl // TestUpdateGaugeMetric follows same pattern
func TestUpdateGaugeMetric(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}
	type metric struct {
		name  string
		value float64
	}

	cases := []struct {
		name        string
		metric      metric
		ExpectedErr error
	}{
		{
			name: "updating_gauge_metric:OK",
			metric: metric{
				name:  "test",
				value: 20561.357,
			},
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			_, actualErr := db.UpdateGaugeMetric(&cfg, tc.metric.name, tc.metric.value)
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

func TestGetGaugeMetric(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}
	type metric struct {
		name  string
		value float64
	}

	cases := []struct {
		name        string
		metric      metric
		ExpectedErr error
	}{
		{
			name: "get_gauge_metric:OK",
			metric: metric{
				name:  "test",
				value: 20561.357,
			},
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			v, _, actualErr := db.GetGaugeMetric(&cfg, tc.metric.name)
			if v != tc.metric.value {
				t.Error("returned value does not match.")
				return
			}
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

//nolint:dupl // storage integration tests follow same pattern
func TestUpdateCounterMetric(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}
	type metric struct {
		name  string
		value int64
	}

	cases := []struct {
		name        string
		metric      metric
		ExpectedErr error
	}{
		{
			name: "updating_counter_metric:OK",
			metric: metric{
				name:  "test",
				value: 2056,
			},
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			_, actualErr := db.UpdateCounterMetric(&cfg, tc.metric.name, tc.metric.value)
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

func TestGetCounterMetric(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}
	type metric struct {
		name  string
		value int64
	}

	cases := []struct {
		name        string
		metric      metric
		ExpectedErr error
	}{
		{
			name: "get_counter_metric:OK",
			metric: metric{
				name:  "test",
				value: 2056,
			},
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			v, _, actualErr := db.GetCounterMetric(&cfg, tc.metric.name)
			if v != tc.metric.value {
				t.Error("returned value does not match.")
				return
			}
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

func TestUpdateBatch(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}
	var f = 2562434.353251
	var i int64 = 25032

	cases := []struct {
		name        string
		gauge       models.Metrics
		counter     models.Metrics
		ExpectedErr error
	}{
		{
			name: "updating_batch_metrics:OK",
			gauge: models.Metrics{
				{
					Value: &f,
					ID:    "testgauge01",
					MType: "gauge",
				},
				{
					Value: &f,
					ID:    "testgauge02",
					MType: "gauge",
				},
			},
			counter: models.Metrics{
				{
					Delta: &i,
					ID:    "testcounter01",
					MType: "counter",
				},
				{
					Delta: &i,
					ID:    "testcounter02",
					MType: "counter",
				},
			},
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			actualErr := db.UpdateBatch(&cfg, tc.gauge, tc.counter)
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

func TestGetAllMetrics(t *testing.T) {
	dsn := getDSN()
	if err := runMigrations(dsn); err != nil {
		t.Errorf("failed to run migrations using dsn %s: %v", dsn, err)
		return
	}

	cfg := models.Config{
		ContextTimeout: 10,
	}

	cases := []struct {
		name        string
		ExpectedErr error
	}{
		{
			name:        "get_all_metrics:OK",
			ExpectedErr: nil,
		},
	}

	db, err := NewPostgresStorage(dsn)
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	for i, tc := range cases {
		i, tc := i, tc

		t.Run(fmt.Sprintf("test #%d: %s", i, tc.name), func(t *testing.T) {
			_, _, actualErr := db.GetAllMetrics(&cfg)
			if err := checkErrors(actualErr, tc.ExpectedErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}

func checkErrors(actual error, expected error) error {
	if actual == nil && expected == nil {
		return nil
	}
	if expected == nil {
		return fmt.Errorf("expected a nil error, but actually got %w", actual)
	}
	if actual == nil {
		return fmt.Errorf("expected an error %w, but actually got nil", expected)
	}
	if actual.Error() != expected.Error() {
		return fmt.Errorf("expected error %w, got %w", expected, actual)
	}
	return nil
}
