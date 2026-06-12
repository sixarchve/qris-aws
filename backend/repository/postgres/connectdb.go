package postgres

import (
	"log"
	"qris-latency-optimizer/domain/entity"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	var err error
	dsn := LoadDatabaseConfig()

	// Retry loop for Postgres connection robustness
	for i := 1; i <= 5; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("PostgreSQL connection failed (attempt %d/5): %v. Retrying in 2s...", i, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic(err)
	}

	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		panic(err)
	}

	var c entity.Merchant
	var d entity.Transaction

	if err := DB.AutoMigrate(&c, &d); err != nil {
		panic(err)
	}

	seedMerchants()
}

func seedMerchants() {
	merchants := []entity.Merchant{
		{QRID: "TEST001", MerchantName: "Kantin FILKOM UB", IsActive: true},
		{QRID: "TEST002", MerchantName: "TESTING STORE", IsActive: true},
	}

	for _, merchant := range merchants {
		if err := DB.Where("qr_id = ?", merchant.QRID).FirstOrCreate(&merchant).Error; err != nil {
			panic(err)
		}
	}
}
