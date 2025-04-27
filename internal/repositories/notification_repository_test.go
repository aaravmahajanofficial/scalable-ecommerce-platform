package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func setupNotificationRepoTest(t *testing.T) (repository.NotificationRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err, "Failed to create sqlmock")

	t.Cleanup(func() {
		db.Close()
	})

	repo := repository.NewNotificationRepo(db)
	require.NotNil(t, repo, "NewNotificationRepo should return a non-nil repository")

	return repo, mock
}

func TestNewNotificationRepo(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewNotificationRepo(db)
	assert.NotNil(t, repo, "NewNotificationRepo should return a non-nil repository")
}

func TestNotificationRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateNotification", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notification := &models.Notification{
				ID:        uuid.New(),
				Type:      "email",
				Recipient: "test@example.com",
				Subject:   "Test Subject",
				Content:   "Test Content",
				Status:    models.StatusPending,
				Metadata:  json.RawMessage(`{"key":"value"}`),
			}

			expectedSQL := regexp.QuoteMeta(`
                INSERT INTO notifications (id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
            `)

			// Expect the ExecContext call
			mock.ExpectExec(expectedSQL).
				WithArgs(notification.ID, notification.Type, notification.Recipient, notification.Subject, notification.Content, notification.Status, notification.ErrorMessage, notification.Metadata).
				WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate 1 row inserted

			// Act
			err := repo.CreateNotification(ctx, notification)

			// Assert
			require.NoError(t, err, "CreateNotification should succeed")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notification := &models.Notification{
				ID:        uuid.New(),
				Type:      "sms",
				Recipient: "+1234567890",
				Subject:   "SMS Subject",
				Content:   "SMS Content",
				Status:    models.StatusPending,
			}
			dbError := errors.New("database insertion error")

			expectedSQL := regexp.QuoteMeta(`
                INSERT INTO notifications (id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
            `)

			// Expect the ExecContext call to fail
			mock.ExpectExec(expectedSQL).
				WithArgs(notification.ID, notification.Type, notification.Recipient, notification.Subject, notification.Content, notification.Status, notification.ErrorMessage, notification.Metadata).
				WillReturnError(dbError)

			// Act
			err := repo.CreateNotification(ctx, notification)

			// Assert
			require.Error(t, err, "CreateNotification should return an error")
			assert.ErrorIs(t, err, dbError, "Returned error should wrap the original database error")
			assert.Contains(t, err.Error(), "failed to create notification", "Error message should indicate creation failure")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})

	t.Run("GetNotificationById", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()
			now := time.Now()
			metadataJSON := json.RawMessage(`{"order_id":"123"}`)
			expectedNotification := &models.Notification{
				ID:        notificationID,
				Type:      "email",
				Recipient: "found@example.com",
				Subject:   "Found Subject",
				Content:   "Found Content",
				Status:    models.StatusSent,
				Metadata:  metadataJSON,
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			}

			expectedSQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                WHERE id = $1
            `)

			// Mock the database call for successful retrieval
			rows := sqlmock.NewRows([]string{"id", "type", "recipient", "subject", "content", "status", "error_message", "metadata", "created_at", "updated_at"}).
				AddRow(expectedNotification.ID, expectedNotification.Type, expectedNotification.Recipient, expectedNotification.Subject, expectedNotification.Content, expectedNotification.Status, expectedNotification.ErrorMessage, []byte(expectedNotification.Metadata), expectedNotification.CreatedAt, expectedNotification.UpdatedAt)
			mock.ExpectQuery(expectedSQL).
				WithArgs(notificationID).
				WillReturnRows(rows)

			// Act
			result, err := repo.GetNotificationById(ctx, notificationID)

			// Assert
			require.NoError(t, err, "GetNotificationById should succeed")
			assert.Equal(t, expectedNotification, result, "Returned notification should match the expected one")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - NotFound", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()

			expectedSQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                WHERE id = $1
            `)

			// Mock the database call to return sql.ErrNoRows
			mock.ExpectQuery(expectedSQL).
				WithArgs(notificationID).
				WillReturnError(sql.ErrNoRows)

			// Act
			result, err := repo.GetNotificationById(ctx, notificationID)

			// Assert
			require.Error(t, err, "GetNotificationById should return an error when not found")
			assert.ErrorIs(t, err, sql.ErrNoRows, "Returned error should wrap sql.ErrNoRows")
			assert.Contains(t, err.Error(), "failed to create notification", "Error message should indicate failure (check implementation for accuracy)")
			assert.NotNil(t, result, "Returned notification should be non-nil (current behavior)")
			assert.Equal(t, models.Notification{}, *result, "Returned notification should be zero value (current behavior)")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Scan Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()

			expectedSQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                WHERE id = $1
            `)

			// Mock the database call with incorrect row data to cause a scan error
			rows := sqlmock.NewRows([]string{"id"}).AddRow("not-a-uuid")
			mock.ExpectQuery(expectedSQL).
				WithArgs(notificationID).
				WillReturnRows(rows)

			// Act
			result, err := repo.GetNotificationById(ctx, notificationID)

			// Assert
			// Similar to NotFound, this tests the current behavior of returning an empty struct and error.
			require.Error(t, err, "GetNotificationById should return an error on scan error")
			assert.NotErrorIs(t, err, sql.ErrNoRows, "Error should not be ErrNoRows")
			// Check the error message based on current implementation.
			assert.Contains(t, err.Error(), "failed to create notification", "Error message should indicate failure (check implementation for accuracy)")
			assert.NotNil(t, result, "Returned notification should be non-nil (current behavior)")
			assert.Equal(t, models.Notification{}, *result, "Returned notification should be zero value (current behavior)")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})

	t.Run("UpdateNotificationStatus", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()
			newStatus := models.StatusSent
			errorMsg := ""

			expectedSQL := regexp.QuoteMeta(`
                UPDATE notifications SET status = $1, error_message = $2, updated_at = $3
                WHERE id = $4
            `)

			// Expect the ExecContext call to succeed and affect 1 row
			mock.ExpectExec(expectedSQL).
				WithArgs(newStatus, errorMsg, sqlmock.AnyArg(), notificationID). // Use AnyArg for time.Now()
				WillReturnResult(sqlmock.NewResult(0, 1))                        // 1 row affected

			// Act
			err := repo.UpdateNotificationStatus(ctx, notificationID, newStatus, errorMsg)

			// Assert
			require.NoError(t, err, "UpdateNotificationStatus should succeed")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Not Found", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()
			newStatus := models.StatusFailed
			errorMsg := "Service unavailable"

			expectedSQL := regexp.QuoteMeta(`
                UPDATE notifications SET status = $1, error_message = $2, updated_at = $3
                WHERE id = $4
            `)

			// Expect the ExecContext call to succeed but affect 0 rows
			mock.ExpectExec(expectedSQL).
				WithArgs(newStatus, errorMsg, sqlmock.AnyArg(), notificationID).
				WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows affected

			// Act
			err := repo.UpdateNotificationStatus(ctx, notificationID, newStatus, errorMsg)

			// Assert
			require.Error(t, err, "UpdateNotificationStatus should return an error when not found")
			assert.Contains(t, err.Error(), fmt.Sprintf("notification not found: %s", notificationID), "Error message should indicate not found")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Exec Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()
			newStatus := models.StatusSent
			errorMsg := ""
			dbError := errors.New("exec error")

			expectedSQL := regexp.QuoteMeta(`
                UPDATE notifications SET status = $1, error_message = $2, updated_at = $3
                WHERE id = $4
            `)

			// Expect the ExecContext call to fail
			mock.ExpectExec(expectedSQL).
				WithArgs(newStatus, errorMsg, sqlmock.AnyArg(), notificationID).
				WillReturnError(dbError)

			// Act
			err := repo.UpdateNotificationStatus(ctx, notificationID, newStatus, errorMsg)

			// Assert
			require.Error(t, err, "UpdateNotificationStatus should return an error on exec failure")
			assert.ErrorIs(t, err, dbError, "Returned error should wrap the original database error")
			assert.Contains(t, err.Error(), "failed to update the notification status", "Error message should indicate update failure")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Rows Affected Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			notificationID := uuid.New()
			newStatus := models.StatusSent
			errorMsg := ""
			rowsAffectedError := errors.New("rows affected error")

			expectedSQL := regexp.QuoteMeta(`
                UPDATE notifications SET status = $1, error_message = $2, updated_at = $3
                WHERE id = $4
            `)

			// Expect the ExecContext call to succeed but RowsAffected() to fail
			mock.ExpectExec(expectedSQL).
				WithArgs(newStatus, errorMsg, sqlmock.AnyArg(), notificationID).
				WillReturnResult(sqlmock.NewErrorResult(rowsAffectedError))

			// Act
			err := repo.UpdateNotificationStatus(ctx, notificationID, newStatus, errorMsg)

			// Assert
			require.Error(t, err, "UpdateNotificationStatus should return an error on RowsAffected failure")
			assert.ErrorIs(t, err, rowsAffectedError, "Returned error should wrap the RowsAffected error")
			assert.Contains(t, err.Error(), "failed to get updated rows", "Error message should indicate RowsAffected failure")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})

	t.Run("ListNotifications", func(t *testing.T) {
		page := 1
		size := 10
		offset := (page - 1) * size

		t.Run("SuccessWithResults", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			totalCount := 5 // Example total count
			now := time.Now()
			metadataJSON := json.RawMessage(`{"key":"val"}`)
			expectedNotifications := []*models.Notification{
				{ID: uuid.New(), Type: "email", Recipient: "n1@example.com", Subject: "S1", Content: "C1", Status: models.StatusSent, Metadata: metadataJSON, CreatedAt: now.Add(-time.Minute), UpdatedAt: now},
				{ID: uuid.New(), Type: "sms", Recipient: "+111", Subject: "S2", Content: "C2", Status: models.StatusPending, Metadata: metadataJSON, CreatedAt: now.Add(-2 * time.Minute), UpdatedAt: now.Add(-time.Minute)},
			}

			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Expect list query
			listQuerySQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
            `)
			rows := sqlmock.NewRows([]string{"id", "type", "recipient", "subject", "content", "status", "metadata", "error_message", "created_at", "updated_at"})
			for _, n := range expectedNotifications {
				rows.AddRow(n.ID, n.Type, n.Recipient, n.Subject, n.Content, n.Status, []byte(n.Metadata), n.ErrorMessage, n.CreatedAt, n.UpdatedAt)
			}
			mock.ExpectQuery(listQuerySQL).
				WithArgs(size, offset).
				WillReturnRows(rows)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.NoError(t, err, "ListNotifications should succeed")
			assert.Equal(t, totalCount, total, "Total count should match expected")
			assert.Equal(t, expectedNotifications, results, "Returned notifications should match expected")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("SuccessNoResults", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			totalCount := 0 // No notifications exist

			// Expect count query
			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Expect list query (will return no rows)
			listQuerySQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
            `)
			rows := sqlmock.NewRows([]string{"id", "type", "recipient", "subject", "content", "status", "metadata", "error_message", "created_at", "updated_at"}) // Empty rows
			mock.ExpectQuery(listQuerySQL).
				WithArgs(size, offset).
				WillReturnRows(rows)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.NoError(t, err, "ListNotifications should succeed even with no results")
			assert.Equal(t, totalCount, total, "Total count should be 0")
			assert.Empty(t, results, "Returned notifications slice should be empty")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Count Query Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			dbError := errors.New("count query failed")

			// Expect count query to fail
			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnError(dbError)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.Error(t, err, "ListNotifications should return an error on count query failure")
			assert.ErrorIs(t, err, dbError, "Returned error should be the count query error")
			assert.Nil(t, results, "Results should be nil on error")
			assert.Zero(t, total, "Total should be zero on error")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - List Query Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			totalCount := 10
			dbError := errors.New("list query failed")

			// Expect count query to succeed
			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Expect list query to fail
			listQuerySQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
            `)
			mock.ExpectQuery(listQuerySQL).
				WithArgs(size, offset).
				WillReturnError(dbError)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.Error(t, err, "ListNotifications should return an error on list query failure")
			assert.ErrorIs(t, err, dbError, "Returned error should wrap the list query error")
			assert.Contains(t, err.Error(), "failed to query notifications", "Error message should indicate list query failure")
			assert.Nil(t, results, "Results should be nil on error")
			assert.Zero(t, total, "Total should be zero on error")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Scan Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			totalCount := 1
			// Expect count query
			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Expect list query with bad data
			listQuerySQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
            `)
			// Return a row that will cause a scan error (e.g., wrong type for ID)
			rows := sqlmock.NewRows([]string{"id", "type", "recipient", "subject", "content", "status", "metadata", "error_message", "created_at", "updated_at"}).
				AddRow("not-a-uuid", "email", "r", "s", "c", "p", []byte("{}"), "", time.Now(), time.Now())
			mock.ExpectQuery(listQuerySQL).
				WithArgs(size, offset).
				WillReturnRows(rows)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.Error(t, err, "ListNotifications should return an error on scan failure")
			assert.Contains(t, err.Error(), "failed to scan notifications", "Error message should indicate scan failure")
			assert.Nil(t, results, "Results should be nil on error")
			assert.Zero(t, total, "Total should be zero on error")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})

		t.Run("Failure - Rows Iteration Error", func(t *testing.T) {
			// Arrange
			repo, mock := setupNotificationRepoTest(t)
			totalCount := 1
			rowsErr := errors.New("rows iteration error")

			// Expect count query
			countQuerySQL := regexp.QuoteMeta(`SELECT COUNT(*) FROM notifications`)
			mock.ExpectQuery(countQuerySQL).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalCount))

			// Expect list query
			listQuerySQL := regexp.QuoteMeta(`
                SELECT id, type, recipient, subject, content, status, error_message, metadata, created_at, updated_at
                FROM notifications
                ORDER BY created_at DESC
                LIMIT $1 OFFSET $2
            `)
			// Return rows that will have an error during iteration (after Next() returns false)
			rows := sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()).RowError(0, rowsErr)
			mock.ExpectQuery(listQuerySQL).
				WithArgs(size, offset).
				WillReturnRows(rows)

			// Act
			results, total, err := repo.ListNotifications(ctx, page, size)

			// Assert
			require.Error(t, err, "ListNotifications should return an error on rows iteration error")
			assert.ErrorIs(t, err, rowsErr, "Returned error should wrap the rows iteration error")
			assert.Contains(t, err.Error(), "error iterating over the rows", "Error message should indicate iteration error")
			assert.Nil(t, results, "Results should be nil on error")
			assert.Zero(t, total, "Total should be zero on error")
			assert.NoError(t, mock.ExpectationsWereMet(), "SQL mock expectations were not met")
		})
	})
}
