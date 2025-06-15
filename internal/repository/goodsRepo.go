package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/yangirxd/goods-service/internal/models"
)

type GoodsRepository struct {
	db *sql.DB
}

func NewGoodsRepository(db *sql.DB) *GoodsRepository {
	return &GoodsRepository{db: db}
}

func (r *GoodsRepository) Create(ctx context.Context, good *models.GoodCreate) (*models.Good, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var maxPriority int
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(priority), 0) 
		FROM goods 
		WHERE project_id = $1
	`, good.ProjectID).Scan(&maxPriority)
	if err != nil {
		return nil, fmt.Errorf("get max priority: %w", err)
	}

	newGood := &models.Good{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO goods (project_id, name, description, priority) 
		VALUES ($1, $2, $3, $4)
		RETURNING id, project_id, name, description, priority, removed, created_at
	`, good.ProjectID, good.Name, good.Description, maxPriority+1).
		Scan(&newGood.ID, &newGood.ProjectID, &newGood.Name, &newGood.Description,
			&newGood.Priority, &newGood.Removed, &newGood.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert good: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return newGood, nil
}

func (r *GoodsRepository) Get(ctx context.Context, id int64) (*models.Good, error) {
	good := &models.Good{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, description, priority, removed, created_at
		FROM goods
		WHERE id = $1 AND removed = false
	`, id).Scan(&good.ID, &good.ProjectID, &good.Name, &good.Description,
		&good.Priority, &good.Removed, &good.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select good: %w", err)
	}

	return good, nil
}

func (r *GoodsRepository) Update(ctx context.Context, id int64, update *models.GoodUpdate) (*models.Good, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	good := &models.Good{}
	err = tx.QueryRowContext(ctx, `
		SELECT id, project_id, name, description, priority, removed, created_at
		FROM goods
		WHERE id = $1 AND removed = false
		FOR UPDATE
	`, id).Scan(&good.ID, &good.ProjectID, &good.Name, &good.Description,
		&good.Priority, &good.Removed, &good.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select good for update: %w", err)
	}

	if update.Name != nil {
		good.Name = *update.Name
	}
	if update.Description != nil {
		good.Description = *update.Description
	}

	err = tx.QueryRowContext(ctx, `
		UPDATE goods
		SET name = $1, description = $2
		WHERE id = $3
		RETURNING id, project_id, name, description, priority, removed, created_at
	`, good.Name, good.Description, id).
		Scan(&good.ID, &good.ProjectID, &good.Name, &good.Description,
			&good.Priority, &good.Removed, &good.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("update good: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return good, nil
}

func (r *GoodsRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE goods
		SET removed = true
		WHERE id = $1 AND removed = false
	`, id)
	if err != nil {
		return fmt.Errorf("delete good: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if affected == 0 {
		return nil
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *GoodsRepository) List(ctx context.Context, limit, offset int) ([]*models.Good, int, int, error) { // Получаем общее количество записей и количество удалённых
	var total, removed int
	err := r.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE removed = true) as removed
		FROM goods
	`).Scan(&total, &removed)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("count goods: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, description, priority, removed, created_at
		FROM goods
		WHERE removed = false
		ORDER BY priority
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("select goods: %w", err)
	}
	defer rows.Close()

	var goods []*models.Good
	for rows.Next() {
		good := &models.Good{}
		err := rows.Scan(&good.ID, &good.ProjectID, &good.Name, &good.Description,
			&good.Priority, &good.Removed, &good.CreatedAt)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("scan good: %w", err)
		}
		goods = append(goods, good)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("iterate goods: %w", err)
	}

	return goods, total, removed, nil
}

func (r *GoodsRepository) Reprioritize(ctx context.Context, id int64, projectID int64, newPriority int) ([]*models.Good, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var currentPriority int
	err = tx.QueryRowContext(ctx, `
		SELECT priority 
		FROM goods 
		WHERE id = $1 AND project_id = $2 AND removed = false
		FOR UPDATE
	`, id, projectID).Scan(&currentPriority)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get current priority: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE goods
		SET priority = $3
		WHERE id = $1
		AND project_id = $2
		AND removed = false
	`, id, projectID, newPriority)
	if err != nil {
		return nil, fmt.Errorf("update priority: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		WITH ordered_goods AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY priority) as rn
			FROM goods
			WHERE project_id = $1
			AND removed = false
			AND priority >= $4
			AND id != $3
		)
		UPDATE goods g
		SET priority = $2 + og.rn
		FROM ordered_goods og
		WHERE g.id = og.id
	`, projectID, newPriority, id, currentPriority)

	if err != nil {
		return nil, fmt.Errorf("update priorities: %w", err)
	}

	// Получаем список всех обновлённых товаров
	rows, err := tx.QueryContext(ctx, `
		SELECT id, project_id, name, description, priority, removed, created_at
		FROM goods
		WHERE project_id = $1 AND removed = false
		AND priority >= $2
		ORDER BY priority
	`, projectID, newPriority)
	if err != nil {
		return nil, fmt.Errorf("select updated goods: %w", err)
	}
	defer rows.Close()

	var updatedGoods []*models.Good
	for rows.Next() {
		good := &models.Good{}
		err := rows.Scan(&good.ID, &good.ProjectID, &good.Name, &good.Description,
			&good.Priority, &good.Removed, &good.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan good: %w", err)
		}
		updatedGoods = append(updatedGoods, good)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate goods: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return updatedGoods, nil
}
