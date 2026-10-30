package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddAlbumUploadedImage, downAddAlbumUploadedImage)
}

func upAddAlbumUploadedImage(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `ALTER TABLE album ADD COLUMN uploaded_image VARCHAR(255) DEFAULT ''`)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `ALTER TABLE album ADD COLUMN cover_changed_at DATETIME`)
	return err
}

func downAddAlbumUploadedImage(ctx context.Context, tx *sql.Tx) error {
	return nil
}
