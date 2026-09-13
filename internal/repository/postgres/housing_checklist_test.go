package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHousingChecklistPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run isolated-schema PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := pgx.Identifier{fmt.Sprintf("checklist_test_%d", time.Now().UnixNano())}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	files, err := filepath.Glob("../../../db/migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations: %v", err)
	}
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	mustID := func(sql string, args ...any) domain.ID {
		t.Helper()
		var id domain.ID
		if err := pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	owner := mustID(`INSERT INTO users(email,role,password_hash) VALUES('owner@test','user','test') RETURNING id::text`)
	viewer := mustID(`INSERT INTO users(email,role,password_hash) VALUES('viewer@test','user','test') RETURNING id::text`)
	outsider := mustID(`INSERT INTO users(email,role,password_hash) VALUES('other@test','user','test') RETURNING id::text`)
	family := mustID(`INSERT INTO families(name,family_type) VALUES('Checklist test','family') RETURNING id::text`)
	foreign := mustID(`INSERT INTO families(name,family_type) VALUES('Other family','family') RETURNING id::text`)
	if _, err := pool.Exec(ctx, `INSERT INTO family_members(family_id,user_id,role) VALUES($1,$2,'owner'),($1,$3,'viewer'),($4,$2,'owner')`, family, owner, viewer, foreign); err != nil {
		t.Fatal(err)
	}
	housing := mustID(`INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) VALUES($1,'Home','bto',100,'cash') RETURNING id::text`, family)
	otherHousing := mustID(`INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type) VALUES($1,'Other','bto',100,'cash') RETURNING id::text`, family)
	repo := NewHousingRepository(pool)
	create := func(name string, order int32, familyID domain.ID) domain.HousingCriterion {
		t.Helper()
		c, e := repo.SaveHousingCriterion(ctx, domain.HousingCriterion{FamilyID: familyID, Name: name, DisplayOrder: order}, owner)
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	location := create("Location", 0, family)
	transport := create("Transport", 1, family)
	other := create("Foreign", 0, foreign)
	rating := int32(5)
	answer := domain.HousingAnswer{HousingID: housing, CriterionID: location.ID, State: "complete", Rating: &rating, Notes: "Near family"}
	if _, err = repo.SaveHousingAnswer(ctx, answer, owner); err != nil {
		t.Fatal(err)
	}
	for _, actor := range []domain.ID{viewer, outsider} {
		if _, err = repo.SaveHousingAnswer(ctx, answer, actor); err == nil {
			t.Fatal("unauthorized answer saved")
		}
		if _, err = repo.SaveHousingCriterion(ctx, domain.HousingCriterion{FamilyID: family, Name: "Denied"}, actor); !errors.Is(err, apperror.ErrForbidden) {
			t.Fatalf("unauthorized criterion: %v", err)
		}
		if err = repo.DeleteHousingCriterion(ctx, family, location.ID, actor); !errors.Is(err, apperror.ErrForbidden) {
			t.Fatalf("unauthorized delete: %v", err)
		}
		if err = repo.ReorderHousingCriteria(ctx, family, []domain.ID{transport.ID, location.ID}, actor); !errors.Is(err, apperror.ErrForbidden) {
			t.Fatalf("unauthorized reorder: %v", err)
		}
	}
	if _, err = repo.ListHousingCriteria(ctx, family, outsider); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("outsider read: %v", err)
	}
	if _, err = repo.ListHousingAnswers(ctx, family, outsider); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("outsider answers: %v", err)
	}
	if _, err = repo.ListHousingCriteria(ctx, family, viewer); err != nil {
		t.Fatal(err)
	}
	cross := answer
	cross.CriterionID = other.ID
	if _, err = repo.SaveHousingAnswer(ctx, cross, owner); err == nil {
		t.Fatal("cross-family answer accepted")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO housing_checklist_answers(housing_id,criterion_id,family_id) VALUES($1,$2,$3)`, housing, other.ID, family); err == nil {
		t.Fatal("database allowed cross-family answer")
	}
	if err = repo.ReorderHousingCriteria(ctx, family, []domain.ID{transport.ID, location.ID}, owner); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]domain.ID{{location.ID}, {location.ID, location.ID}, {location.ID, other.ID}} {
		if err = repo.ReorderHousingCriteria(ctx, family, ids, owner); !errors.Is(err, apperror.ErrInvalidArgument) {
			t.Fatalf("invalid reorder: %v", err)
		}
	}
	criteria, err := repo.ListHousingCriteria(ctx, family, viewer)
	if err != nil || criteria[0].ID != transport.ID || criteria[1].DisplayOrder != 1 {
		t.Fatalf("order: %+v %v", criteria, err)
	}
	weight := 3.0
	transport.Weight = &weight
	transport.Name = "Transport access"
	transport.DisplayOrder = 1
	if _, err = repo.SaveHousingCriterion(ctx, transport, owner); err != nil {
		t.Fatal(err)
	}
	rating1 := int32(1)
	if _, err = repo.SaveHousingAnswer(ctx, domain.HousingAnswer{HousingID: housing, CriterionID: transport.ID, State: "pending", Rating: &rating1}, owner); err != nil {
		t.Fatal(err)
	}
	options, err := repo.ListHousingOptions(ctx, family, viewer)
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range options {
		if option.ID == housing {
			if option.Evaluation.Score == nil || *option.Evaluation.Score != 2 || option.Evaluation.Completed != 1 {
				t.Fatalf("weighted score: %+v", option.Evaluation)
			}
		} else if option.Evaluation.Score != nil {
			t.Fatal("answers leaked to another home")
		}
	}
	if err = repo.DeleteHousingCriterion(ctx, family, location.ID, owner); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SaveHousingAnswer(ctx, answer, owner); err == nil {
		t.Fatal("retired criterion accepted new answer")
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM housing_checklist_answers WHERE criterion_id=$1`, location.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("history lost: %d %v", count, err)
	}
	got, err := repo.GetHousingOption(ctx, housing, viewer)
	if err != nil || got.Evaluation.Total != 1 || *got.Evaluation.Score != 1 {
		t.Fatalf("retirement score: %+v %v", got.Evaluation, err)
	}
	if err = repo.DeleteHousingOption(ctx, housing, owner); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM housing_checklist_answers WHERE housing_id=$1`, housing).Scan(&count); err != nil || count != 0 {
		t.Fatalf("answers not cascaded: %d %v", count, err)
	}
	if _, err = repo.GetHousingOption(ctx, otherHousing, viewer); err != nil {
		t.Fatalf("unrelated home lost: %v", err)
	}
	down, err := os.ReadFile("../../../db/migrations/000006_housing_checklist.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	up, err := os.ReadFile("../../../db/migrations/000006_housing_checklist.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply: %v", err)
	}
}
