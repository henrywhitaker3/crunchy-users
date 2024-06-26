package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/henrywhitaker3/crunchy-users/internal/k8s"
	"github.com/stretchr/testify/mock"
)

func testSuperuser() k8s.ClusterSuperuser {
	return k8s.ClusterSuperuser{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "postgres",
		Password: "postgres",
		Database: "postgres",
	}
}

type mockProcessor struct {
	mock.Mock
}

func (m *mockProcessor) UserExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	args := m.Called(ctx, db, name)
	return args.Bool(0), args.Error(1)
}

func (m *mockProcessor) UserIsOwner(ctx context.Context, db *sql.DB, cluster, user, database string) (bool, error) {
	args := m.Called(ctx, db, cluster, user, database)
	return args.Bool(0), args.Error(1)
}

func (m *mockProcessor) DatabaseExists(ctx context.Context, db *sql.DB, cluster string, database string) (bool, error) {
	args := m.Called(ctx, db, cluster, database)
	return args.Bool(0), args.Error(1)
}

func (m *mockProcessor) MakeUserOwner(ctx context.Context, db *sql.DB, database, user string) error {
	args := m.Called(ctx, db, database, user)
	return args.Error(0)
}

func (m *mockProcessor) ExtensionExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	args := m.Called(ctx, db, name)
	return args.Bool(0), args.Error(1)
}

func (m *mockProcessor) CreateExtension(ctx context.Context, db *sql.DB, name string, cascade bool) error {
	args := m.Called(ctx, db, name, cascade)
	return args.Error(0)
}

func setMockProcessor(p Processor) {
	NewProcessor = func() Processor {
		return p
	}
}

func TestItStopsWhenUserDoesNotExistAndNoError(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(false, nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "DatabaseExists")
	m.AssertNotCalled(t, "UserIsOwner")
	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItStopsWhenUserDoesNotExistAndError(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, errors.New("bongo"))

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "DatabaseExists")
	m.AssertNotCalled(t, "UserIsOwner")
	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItChecksAndStopsDatabaseExistsWhenUserExistsNoError(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(false, nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "UserIsOwner")
	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItChecksAndStopsDatabaseExistsWhenUserExistsError(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, errors.New("bongo"))

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "UserIsOwner")
	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItChecksAndStopsWhenUserIsOwnerForSingleDatabasesNoErrors(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(true, nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItChecksAndStopsWhenUserIsOwnerForSingleDatabasesWithErrors(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(false, errors.New("bongo"))

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "MakeUserOwner")
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItMakesTheUserTheOwnerNoErrors(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(false, nil)
	m.On("MakeUserOwner", mock.Anything, mock.Anything, "bongo", "bongo").Return(nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
	})

	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItMakesUserOwnerOfMultipleDatabases(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bingo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(false, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bingo").Return(false, nil)
	m.On("MakeUserOwner", mock.Anything, mock.Anything, "bongo", "bongo").Return(nil)
	m.On("MakeUserOwner", mock.Anything, mock.Anything, "bingo", "bongo").Return(nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo", "bingo"},
			},
		},
	})

	m.AssertNumberOfCalls(t, "DatabaseExists", 2)
	m.AssertNumberOfCalls(t, "UserIsOwner", 2)
	m.AssertNumberOfCalls(t, "MakeUserOwner", 2)
	m.AssertNotCalled(t, "ExtensionExists")
	m.AssertNotCalled(t, "CreateExtension")
}

func TestItDoesntCreateExtensionsWhenTheyExist(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(true, nil)
	m.On("ExtensionExists", mock.Anything, mock.Anything, "vector").Return(true, nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
		Extensions: map[string][]k8s.DatabaseExtension{
			"bongo": {
				{
					Database:  "bongo",
					Extension: "vector",
					Cascade:   true,
				},
			},
		},
	})

	m.AssertNotCalled(t, "CreateExtension")
}

func TestItCreatesExtensionsWhenTheyDontExist(t *testing.T) {
	m := &mockProcessor{}
	setMockProcessor(m)

	m.On("UserExists", mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("DatabaseExists", mock.Anything, mock.Anything, mock.Anything, "bongo").Return(true, nil)
	m.On("UserIsOwner", mock.Anything, mock.Anything, mock.Anything, "bongo", "bongo").Return(true, nil)
	m.On("ExtensionExists", mock.Anything, mock.Anything, "vector").Return(false, nil)
	m.On("CreateExtension", mock.Anything, mock.Anything, "vector", true).Return(nil)

	HandleCluster(context.Background(), k8s.ClusterResult{
		Name:      "test",
		Namespace: "test",
		Superuser: testSuperuser(),
		Users: []k8s.ClusterUser{
			{
				Name:      "bongo",
				Databases: []string{"bongo"},
			},
		},
		Extensions: map[string][]k8s.DatabaseExtension{
			"bongo": {
				{
					Database:  "bongo",
					Extension: "vector",
					Cascade:   true,
				},
			},
		},
	})
}
