package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserRepo(db)
	assert.NotNil(t, repo, "NewUserRepo should return a non-nil repository")
}

func TestUserRepository(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewUserRepo(db)
	ctx := context.Background()

	t.Run("CreateUser_Success", func(t *testing.T) {
		// Arrange
		user := &models.User{
			Email:    "test@example.com",
			Password: "hashedpassword",
			Name:     "Test User",
		}
		now := time.Now()
		newID := uuid.New()

		expectedSQL := regexp.QuoteMeta(`
        INSERT INTO users(email, password, name, created_at, updated_at)
        VALUES($1, $2, $3, NOW(), NOW())
        RETURNING id, created_at, updated_at`)

		// Mock the database call for successful insertion
		mock.ExpectQuery(expectedSQL).
			WithArgs(user.Email, user.Password, user.Name).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(newID, now, now))

		// Act
		err := repo.CreateUser(ctx, user)

		// Assert
		require.NoError(t, err, "CreateUser should not return an error on success")
		assert.Equal(t, newID, user.ID, "User ID should be updated")
		assert.WithinDuration(t, now, user.CreatedAt, time.Second, "User CreatedAt should be updated")
		assert.WithinDuration(t, now, user.UpdatedAt, time.Second, "User UpdatedAt should be updated")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("CreateUser_Error", func(t *testing.T) {
		// Arrange
		user := &models.User{
			Email:    "error@example.com",
			Password: "password",
			Name:     "Error User",
		}
		dbError := errors.New("database insertion error")

		expectedSQL := regexp.QuoteMeta(`
        INSERT INTO users(email, password, name, created_at, updated_at)
        VALUES($1, $2, $3, NOW(), NOW())
        RETURNING id, created_at, updated_at`)

		// Mock the database call to return an error
		mock.ExpectQuery(expectedSQL).
			WithArgs(user.Email, user.Password, user.Name).
			WillReturnError(dbError)

		// Act
		err := repo.CreateUser(ctx, user)

		// Assert
		require.Error(t, err, "CreateUser should return an error")
		assert.Equal(t, dbError, err, "Returned error should match the expected database error")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("GetUserByEmail_Success", func(t *testing.T) {
		// Arrange
		email := "findme@example.com"
		expectedUser := &models.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashedpassword",
			Name:      "Found User",
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		}

		expectedSQL := regexp.QuoteMeta(`SELECT id, email, password, name, created_at, updated_at
              FROM users 
              WHERE email = $1`)

		// Mock the database call for successful retrieval
		rows := sqlmock.NewRows([]string{"id", "email", "password", "name", "created_at", "updated_at"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.Password, expectedUser.Name, expectedUser.CreatedAt, expectedUser.UpdatedAt)
		mock.ExpectQuery(expectedSQL).
			WithArgs(email).
			WillReturnRows(rows)

		// Act
		user, err := repo.GetUserByEmail(ctx, email)

		// Assert
		require.NoError(t, err, "GetUserByEmail should not return an error when user is found")
		assert.Equal(t, expectedUser, user, "Returned user should match the expected user")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("GetUserByEmail_NotFound", func(t *testing.T) {
		// Arrange
		email := "notfound@example.com"

		expectedSQL := regexp.QuoteMeta(`SELECT id, email, password, name, created_at, updated_at
              FROM users 
              WHERE email = $1`)

		// Mock the database call to return sql.ErrNoRows
		mock.ExpectQuery(expectedSQL).
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		// Act
		user, err := repo.GetUserByEmail(ctx, email)

		// Assert
		require.Error(t, err, "GetUserByEmail should return an error when user is not found")
		assert.Equal(t, sql.ErrNoRows, err, "Returned error should be sql.ErrNoRows")
		assert.Nil(t, user, "Returned user should be nil when not found")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("GetUserByEmail_ScanError", func(t *testing.T) {
		// Arrange
		email := "scanerror@example.com"
		expectedSQL := regexp.QuoteMeta(`SELECT id, email, password, name, created_at, updated_at
              FROM users 
              WHERE email = $1`)

		// Mock the database call with incorrect row data to cause a scan error
		rows := sqlmock.NewRows([]string{"id", "email"}).AddRow(uuid.New(), email)
		mock.ExpectQuery(expectedSQL).
			WithArgs(email).
			WillReturnRows(rows)

		// Act
		user, err := repo.GetUserByEmail(ctx, email)

		// Assert
		require.Error(t, err, "GetUserByEmail should return an error on scan error")
		assert.NotEqual(t, sql.ErrNoRows, err, "Error should not be ErrNoRows")
		assert.Nil(t, user, "Returned user should be nil on scan error")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserById_Success", func(t *testing.T) {
		// Arrange
		userID := uuid.New()
		expectedUser := &models.User{
			ID:        userID,
			Email:     "byid@example.com",
			Name:      "User By ID",
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-time.Minute),
		}

		expectedSQL := regexp.QuoteMeta(`
			SELECT id, email, name, created_at, updated_at
			FROM users
			WHERE id = $1
		`)

		// Mock the database call for successful retrieval
		rows := sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.Name, expectedUser.CreatedAt, expectedUser.UpdatedAt)
		mock.ExpectQuery(expectedSQL).
			WithArgs(userID).
			WillReturnRows(rows)

		// Act
		user, err := repo.GetUserById(ctx, userID)

		// Assert
		require.NoError(t, err, "GetUserById should not return an error when user is found")
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Email, user.Email)
		assert.Equal(t, expectedUser.Name, user.Name)
		assert.Equal(t, expectedUser.CreatedAt, user.CreatedAt)
		assert.Equal(t, expectedUser.UpdatedAt, user.UpdatedAt)
		assert.Empty(t, user.Password, "Password should not be populated by GetUserById")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("GetUserById_NotFound", func(t *testing.T) {
		// Arrange
		userID := uuid.New()

		expectedSQL := regexp.QuoteMeta(`
			SELECT id, email, name, created_at, updated_at
			FROM users
			WHERE id = $1
		`)

		// Mock the database call to return sql.ErrNoRows
		mock.ExpectQuery(expectedSQL).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Act
		user, err := repo.GetUserById(ctx, userID)

		// Assert
		require.Error(t, err, "GetUserById should return an error when user is not found")
		assert.Equal(t, "user not found", err.Error(), "Error message should indicate user not found")
		assert.Nil(t, user, "Returned user should be nil when not found")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("GetUserById_ScanError", func(t *testing.T) {
		// Arrange
		userID := uuid.New()
		scanError := errors.New("some other db error")

		expectedSQL := regexp.QuoteMeta(`
			SELECT id, email, name, created_at, updated_at
			FROM users
			WHERE id = $1
		`)

		// Mock the database call to return a generic error
		mock.ExpectQuery(expectedSQL).
			WithArgs(userID).
			WillReturnError(scanError)

		// Act
		user, err := repo.GetUserById(ctx, userID)

		// Assert
		require.Error(t, err, "GetUserById should return an error on a generic database error")
		assert.Equal(t, scanError, err, "Returned error should match the generic database error")
		assert.Nil(t, user, "Returned user should be nil on error")
		assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})
}
