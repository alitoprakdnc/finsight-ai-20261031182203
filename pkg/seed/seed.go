package seed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/yourorg/finSight/internal/config"
	"github.com/yourorg/finSight/internal/models"
	"github.com/yourorg/finSight/internal/repository"
)

// Seed creates demo tenants, users, sample documents and webhook subscriptions.
// It is safe to run multiple times – existing records are left untouched.
func Seed(ctx context.Context) error {
	// -------------------------------------------------------------------------
	// Load configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// -------------------------------------------------------------------------
	// Initialise DB connection
	// -------------------------------------------------------------------------
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	// Auto‑migrate the models we are going to seed.
	if err := db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Document{},
		&models.Webhook{},
		&models.AuditLog{},
	); err != nil {
		return fmt.Errorf("auto‑migrate: %w", err)
	}

	// -------------------------------------------------------------------------
	// Repositories
	// -------------------------------------------------------------------------
	tenantRepo := repository.NewTenantRepository(db)
	userRepo := repository.NewUserRepository(db)
	docRepo := repository.NewDocumentRepository(db)
	webhookRepo := repository.NewWebhookRepository(db)

	// -------------------------------------------------------------------------
	// Seed data inside a transaction – rollback on any error.
	// -------------------------------------------------------------------------
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("starting transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re‑panic after rollback
		}
	}()

	// Helper to commit at the end.
	commit := func() error {
		if err := tx.Commit().Error; err != nil {
			return fmt.Errorf("committing transaction: %w", err)
		}
		return nil
	}

	// -------------------------------------------------------------------------
	// 1️⃣ Demo tenant
	// -------------------------------------------------------------------------
	const demoTenantName = "Acme Corp"
	tenant, err := tenantRepo.GetByName(ctx, demoTenantName)
	if err != nil && err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return fmt.Errorf("query tenant: %w", err)
	}
	if tenant == nil {
		tenant = &models.Tenant{
			ID:        uuid.New(),
			Name:      demoTenantName,
			CreatedAt: time.Now(),
		}
		if err := tenantRepo.Create(ctx, tenant); err != nil {
			tx.Rollback()
			return fmt.Errorf("create tenant: %w", err)
		}
	}

	// -------------------------------------------------------------------------
	// 2️⃣ Demo users (admin, accountant, viewer)
	// -------------------------------------------------------------------------
	type demoUser struct {
		Email string
		Role  string
	}
	users := []demoUser{
		{Email: "admin@acme.local", Role: "admin"},
		{Email: "accountant@acme.local", Role: "accountant"},
		{Email: "viewer@acme.local", Role: "viewer"},
	}
	for _, du := range users {
		u, err := userRepo.GetByEmail(ctx, du.Email)
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("query user %s: %w", du.Email, err)
		}
		if u != nil {
			continue // already exists
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte("ChangeMe123!"), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("hash password: %w", err)
		}
		newUser := &models.User{
			ID:           uuid.New(),
			TenantID:     tenant.ID,
			Email:        du.Email,
			PasswordHash: string(hashed),
			Role:         du.Role,
			CreatedAt:    time.Now(),
		}
		if err := userRepo.Create(ctx, newUser); err != nil {
			tx.Rollback()
			return fmt.Errorf("create user %s: %w", du.Email, err)
		}
	}

	// -------------------------------------------------------------------------
	// 3️⃣ Sample documents
	// -------------------------------------------------------------------------
	sampleDocs := []struct {
		FileName   string
		StorageKey string
	}{
		{FileName: "Invoice_2024-01.pdf", StorageKey: "demo/Invoice_2024-01.pdf"},
		{FileName: "Receipt_2024-02.png", StorageKey: "demo/Receipt_2024-02.png"},
	}
	for _, sd := range sampleDocs {
		// Check existence by storage key
		exists, err := docRepo.ExistsByStorageKey(ctx, sd.StorageKey)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("checking document %s: %w", sd.StorageKey, err)
		}
		if exists {
			continue
		}
		doc := &models.Document{
			ID:          uuid.New(),
			TenantID:    tenant.ID,
			UploadedBy:  "admin@acme.local",
			FileName:    sd.FileName,
			StorageKey:  sd.StorageKey,
			Status:      models.DocumentStatusUploaded,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			ExtractedData: nil, // will be filled by worker later
		}
		if err := docRepo.Create(ctx, doc); err != nil {
			tx.Rollback()
			return fmt.Errorf("create document %s: %w", sd.FileName, err)
		}
	}

	// -------------------------------------------------------------------------
	// 4️⃣ Webhook subscriptions
	// -------------------------------------------------------------------------
	webhookURL := "https://webhook.site/your-demo-endpoint"
	secret, err := generateSecret(32)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("generate webhook secret: %w", err)
	}
	exists, err := webhookRepo.ExistsByURL(ctx, tenant.ID, webhookURL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("checking webhook existence: %w", err)
	}
	if !exists {
		sub := &models.Webhook{
			ID:        uuid.New(),
			TenantID:  tenant.ID,
			URL:       webhookURL,
			Secret:    secret,
			Events:   []string{"document.processed", "document.approved"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := webhookRepo.Create(ctx, sub); err != nil {
			tx.Rollback()
			return fmt.Errorf("create webhook: %w", err)
		}
	}

	// -------------------------------------------------------------------------
	// Commit transaction
	// -------------------------------------------------------------------------
	if err := commit(); err != nil {
		return err
	}

	return nil
}

// generateSecret returns a hex‑encoded random secret of the requested byte length.
func generateSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}