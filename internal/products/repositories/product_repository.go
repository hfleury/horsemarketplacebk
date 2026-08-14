package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
)

type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) (*models.Product, error)
	FindByID(ctx context.Context, id string) (*models.Product, error)
	FindAll(ctx context.Context, filters map[string]any, page, limit int) (items []*models.Product, total int, err error)
	FindByCategory(ctx context.Context, categoryID string, page, limit int) (items []*models.Product, total int, err error)
	FindByField(ctx context.Context, fieldName string, value string) ([]*models.Product, error)
	SearchByFilter(ctx context.Context, categoryID, query string, filter *models.HorseFilter, locationFilter *models.LocationFilter, page, limit int) (items []*models.Product, total int, err error)
	UpdateStatus(ctx context.Context, id string, status models.ProductStatus) error
	Delete(ctx context.Context, id string) error
	// Add Update method later as it's complex
}

type ProductRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewProductRepoPsql(psql db.Database, logger config.Logging) *ProductRepoPsql {
	return &ProductRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *ProductRepoPsql) Create(ctx context.Context, product *models.Product) (*models.Product, error) {
	tx, err := r.psql.BeginTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 1. Insert into products table
	queryProd := `
		INSERT INTO catalog.products (
			id, user_id, category_id, type, status, title, price_sek, description,
			city, area, transaction_type, views_count, latitude, longitude
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 0, $12, $13)
		RETURNING id, created_at, updated_at
	`

	productID := uuid.New()
	if product.ID != uuid.Nil {
		productID = product.ID
	}

	err = tx.QueryRowContext(ctx, queryProd,
		productID, product.UserID, product.CategoryID, product.Type, product.Status,
		product.Title, product.PriceSEK, product.Description, product.City,
		product.Area, product.TransactionType, product.Latitude, product.Longitude,
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to insert product", map[string]any{"error": err.Error()})
		return nil, err
	}

	// 2. Insert specific data based on type
	if err := r.insertSpecificData(ctx, tx, product); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to insert specific product data", map[string]any{"error": err.Error(), "type": product.Type})
		return nil, err
	}

	// 3. Commit
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return product, nil
}

func (r *ProductRepoPsql) insertSpecificData(ctx context.Context, tx *sql.Tx, p *models.Product) error {
	switch p.Type {
	case models.TypeHorse:
		if p.Horse == nil {
			return errors.New("horse data missing")
		}
		q := `INSERT INTO catalog.product_horses (product_id, name, age, year_of_birth, gender, height, breed, color, dressage_level, jump_level, orientation, pedigree)
		      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

		_, err := tx.ExecContext(ctx, q, p.ID, p.Horse.Name, p.Horse.Age, p.Horse.YearOfBirth, p.Horse.Gender, p.Horse.Height, p.Horse.Breed, p.Horse.Color, p.Horse.DressageLevel, p.Horse.JumpLevel, p.Horse.Orientation, p.Horse.Pedigree)
		return err

	case models.TypeVehicle:
		if p.Vehicle == nil {
			return errors.New("vehicle data missing")
		}
		q := `INSERT INTO catalog.product_vehicles (product_id, make, model, year, load_weight, total_weight, condition)
		      VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := tx.ExecContext(ctx, q, p.ID, p.Vehicle.Make, p.Vehicle.Model, p.Vehicle.Year, p.Vehicle.LoadWeight, p.Vehicle.TotalWeight, p.Vehicle.Condition)
		return err

	case models.TypeEquipment:
		if p.Equipment == nil {
			return errors.New("equipment data missing")
		}
		q := `INSERT INTO catalog.product_equipment (product_id, make, model, size, condition, sub_type, boom_width)
		      VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := tx.ExecContext(ctx, q, p.ID, p.Equipment.Make, p.Equipment.Model, p.Equipment.Size, p.Equipment.Condition, p.Equipment.SubType, p.Equipment.BoomWidth)
		return err

	default:
		// For other types like Service, we might only use the main table for now
		return nil
	}
}

// Base query to fetch all fields including joined specific tables
// We limit SELECT * to specific aliases to avoid ambiguous columns
var selectFullProduct = `
	SELECT 
		p.id, p.user_id, p.category_id, p.type, p.status, p.title, p.price_sek, p.description,
		p.city, p.area, p.latitude, p.longitude, p.transaction_type, p.views_count, p.created_at, p.updated_at,
		h.name, h.age, h.year_of_birth, h.gender, h.height, h.breed, h.color, h.dressage_level, h.jump_level, h.orientation, h.pedigree,
		v.make, v.model, v.year, v.load_weight, v.total_weight, v.condition,
		e.make, e.model, e.size, e.condition, e.sub_type, e.boom_width
	FROM catalog.products p
	LEFT JOIN catalog.product_horses h ON p.id = h.product_id
	LEFT JOIN catalog.product_vehicles v ON p.id = v.product_id
	LEFT JOIN catalog.product_equipment e ON p.id = e.product_id
`

func (r *ProductRepoPsql) scanProduct(row interface{ Scan(...any) error }) (*models.Product, error) {
	var p models.Product
	// Pointers for specific fields that might be null
	var (
		// Horse
		hName, hGender, hBreed, hColor, hDressage, hJump, hOrient *string
		hAge, hYOB, hHeight                                       *int
		hPedigree                                                 []byte

		// Vehicle
		vMake, vModel, vCondition *string
		vYear, vLoad, vTotal      *int

		// Equipment
		eMake, eModel, eSize, eCondition, eSubType, eBoom *string
	)

	err := row.Scan(
		&p.ID, &p.UserID, &p.CategoryID, &p.Type, &p.Status, &p.Title, &p.PriceSEK, &p.Description,
		&p.City, &p.Area, &p.Latitude, &p.Longitude, &p.TransactionType, &p.ViewsCount, &p.CreatedAt, &p.UpdatedAt,
		&hName, &hAge, &hYOB, &hGender, &hHeight, &hBreed, &hColor, &hDressage, &hJump, &hOrient, &hPedigree,
		&vMake, &vModel, &vYear, &vLoad, &vTotal, &vCondition,
		&eMake, &eModel, &eSize, &eCondition, &eSubType, &eBoom,
	)
	if err != nil {
		return nil, err
	}

	// Populate specific structs based on Type
	switch p.Type {
	case models.TypeHorse:
		p.Horse = &models.Horse{
			ProductID: p.ID, Name: hName, Age: hAge, YearOfBirth: hYOB, Gender: hGender, Height: hHeight,
			Breed: hBreed, Color: hColor, DressageLevel: hDressage, JumpLevel: hJump, Orientation: hOrient,
		}
		if hPedigree != nil {
			p.Horse.Pedigree = json.RawMessage(hPedigree)
		}
	case models.TypeVehicle:
		p.Vehicle = &models.Vehicle{
			ProductID: p.ID, Make: vMake, Model: vModel, Year: vYear, LoadWeight: vLoad,
			TotalWeight: vTotal, Condition: vCondition,
		}
	case models.TypeEquipment:
		p.Equipment = &models.Equipment{
			ProductID: p.ID, Make: eMake, Model: eModel, Size: eSize, Condition: eCondition,
			SubType: eSubType, BoomWidth: eBoom,
		}
	}

	return &p, nil
}

func (r *ProductRepoPsql) FindByID(ctx context.Context, id string) (*models.Product, error) {
	query := selectFullProduct + ` WHERE p.id = $1`
	row := r.psql.QueryRow(ctx, query, id)
	p, err := r.scanProduct(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *ProductRepoPsql) FindAll(ctx context.Context, filters map[string]any, page, limit int) ([]*models.Product, int, error) {
	// Basic implementation pending more complex filtering logic
	offset := (page - 1) * limit

	var total int
	if err := r.psql.QueryRow(ctx, `SELECT COUNT(*) FROM catalog.products`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := selectFullProduct + ` ORDER BY p.created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.psql.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p, err := r.scanProduct(rows)
		if err != nil {
			continue // Log error?
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepoPsql) FindByCategory(ctx context.Context, categoryID string, page, limit int) ([]*models.Product, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.psql.QueryRow(ctx, `SELECT COUNT(*) FROM catalog.products WHERE category_id = $1`, categoryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := selectFullProduct + ` WHERE p.category_id = $1 ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.psql.Query(ctx, query, categoryID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p, err := r.scanProduct(rows)
		if err != nil {
			continue
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepoPsql) FindByField(ctx context.Context, fieldName string, value string) ([]*models.Product, error) {
	// WARNING: fieldName injection risk if coming from user input.
	// Validate fieldName against allowed list is crucial.
	allowedFields := map[string]bool{
		"model": true, "make": true, "year": true, "breed": true, "gender": true,
	}
	if !allowedFields[strings.ToLower(fieldName)] {
		return nil, errors.New("invalid search field")
	}

	// We need to know which table the field belongs to, or check all?
	// Simplified: Coalesce or check specific type tables.
	// This is tricky with raw SQL and dynamic fields without a huge CASE or dynamic query builder.
	// For "Model", it exists in Vehicle and Equipment.

	// Dynamic construction (careful!)
	// assuming fieldName matches column name exactly for now
	// To be safe, we check if it's in the joined columns.

	// Better approach for "FindByField(Model=X)" type queries:
	var whereClause string
	switch strings.ToLower(fieldName) {
	case "model":
		whereClause = "v.model = $1 OR e.model = $1"
	case "make":
		whereClause = "v.make = $1 OR e.make = $1"
	case "breed":
		whereClause = "h.breed = $1"
	default:
		// Fallback or specific logic
		return nil, fmt.Errorf("search by %s not implemented", fieldName)
	}

	query := selectFullProduct + " WHERE " + whereClause + " ORDER BY p.created_at DESC"

	rows, err := r.psql.Query(ctx, query, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p, err := r.scanProduct(rows)
		if err != nil {
			continue
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepoPsql) SearchByFilter(ctx context.Context, categoryID, query string, filter *models.HorseFilter, locationFilter *models.LocationFilter, page, limit int) ([]*models.Product, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []any

	addCondition := func(clause string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(clause, len(args)))
	}

	if categoryID != "" {
		addCondition("p.category_id = $%d", categoryID)
	}
	if query != "" {
		args = append(args, "%"+query+"%")
		conditions = append(conditions, fmt.Sprintf("(p.description ILIKE $%d OR p.title ILIKE $%d)", len(args), len(args)))
	}
	if filter != nil {
		if filter.Breed != nil && *filter.Breed != "" {
			addCondition("h.breed = $%d", *filter.Breed)
		}
		if filter.Gender != nil && *filter.Gender != "" {
			addCondition("h.gender = $%d", *filter.Gender)
		}
		if filter.Discipline != nil && *filter.Discipline != "" {
			addCondition("h.orientation = $%d", *filter.Discipline)
		}
		if filter.MinAge != nil {
			addCondition("h.age >= $%d", *filter.MinAge)
		}
		if filter.MaxAge != nil {
			addCondition("h.age <= $%d", *filter.MaxAge)
		}
		if filter.MinHeight != nil {
			addCondition("h.height >= $%d", *filter.MinHeight)
		}
		if filter.MaxHeight != nil {
			addCondition("h.height <= $%d", *filter.MaxHeight)
		}
		if filter.MinPrice != nil {
			addCondition("p.price_sek >= $%d", *filter.MinPrice)
		}
		if filter.MaxPrice != nil {
			addCondition("p.price_sek <= $%d", *filter.MaxPrice)
		}
	}
	if locationFilter != nil {
		args = append(args, locationFilter.Lng, locationFilter.Lat, locationFilter.RadiusKm*1000)
		n := len(args)
		conditions = append(conditions, fmt.Sprintf(
			"ST_DWithin(ST_SetSRID(ST_MakePoint(p.longitude, p.latitude), 4326)::geography, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)",
			n-2, n-1, n,
		))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM catalog.products p LEFT JOIN catalog.product_horses h ON p.id = h.product_id" + whereClause
	var total int
	if err := r.psql.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	dataArgs := append(append([]any{}, args...), limit, offset)
	dataQuery := selectFullProduct + whereClause +
		fmt.Sprintf(" ORDER BY p.created_at DESC LIMIT $%d OFFSET $%d", len(dataArgs)-1, len(dataArgs))

	rows, err := r.psql.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []*models.Product
	for rows.Next() {
		p, err := r.scanProduct(rows)
		if err != nil {
			continue
		}
		products = append(products, p)
	}
	return products, total, nil
}

func (r *ProductRepoPsql) UpdateStatus(ctx context.Context, id string, status models.ProductStatus) error {
	query := `UPDATE catalog.products SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.psql.Execute(ctx, query, status, id)
	return err
}

func (r *ProductRepoPsql) Delete(ctx context.Context, id string) error {
	// Virtual delete or real? User said "User can delete yours products (virtual delete)"
	// "Admin can delete all (virtual delete)"
	return r.UpdateStatus(ctx, id, models.StatusDeleted)
}
