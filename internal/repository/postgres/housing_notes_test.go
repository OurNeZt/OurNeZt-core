package postgres

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Called by the isolated-schema integration fixture after a checklist rating exists.
func testHousingNotes(t *testing.T, pool *pgxpool.Pool, housing, family, owner, viewer, outsider domain.ID) {
	t.Helper()
	ctx := context.Background()
	repo := NewHousingRepository(pool)
	before, err := repo.GetHousingOption(ctx, housing, owner)
	if err != nil || before.Notes != "" {
		t.Fatalf("existing home default notes: %v", err)
	}
	answersBefore, err := repo.ListHousingAnswers(ctx, family, owner)
	if err != nil {
		t.Fatal(err)
	}
	notes := "  Agent: call back\nViewing feedback 🏡\n"
	for _, actor := range []domain.ID{owner, viewer, outsider} {
		saved, err := repo.UpdateHousingNotes(ctx, housing, notes, actor)
		if actor == owner {
			if err != nil || saved != notes {
				t.Fatalf("save notes: %v", err)
			}
		} else if err == nil {
			t.Fatal("unauthorized notes update succeeded")
		}
	}
	for _, role := range []string{"admin", "member"} {
		var actor domain.ID
		if err := pool.QueryRow(ctx, `INSERT INTO users(email,role,password_hash) VALUES($1,'user','test') RETURNING id::text`, role+"@notes.test").Scan(&actor); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO family_members(family_id,user_id,role) VALUES($1,$2,$3)`, family, actor, role); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.UpdateHousingNotes(ctx, housing, notes, actor); err != nil {
			t.Fatalf("%s cannot save: %v", role, err)
		}
	}
	if _, err := repo.GetHousingOption(ctx, housing, outsider); err == nil {
		t.Fatal("outsider read notes")
	}
	if _, err := repo.ListHousingOptions(ctx, family, outsider); err == nil {
		t.Fatal("outsider listed notes")
	}
	got, err := repo.GetHousingOption(ctx, housing, viewer)
	if err != nil || got.Notes != notes {
		t.Fatalf("viewer cannot read notes: %v", err)
	}
	if got.PurchasePriceCents != before.PurchasePriceCents || !reflect.DeepEqual(got.Evaluation, before.Evaluation) {
		t.Fatal("notes changed housing inputs or evaluation")
	}
	// A stale detail form must not overwrite a later notes save.
	before.Name = "Edited home"
	before.Notes = "stale text"
	updated, err := repo.UpdateHousingOption(ctx, before, owner)
	if err != nil || updated.Notes != notes {
		t.Fatalf("detail update lost notes: %v", err)
	}
	options, err := repo.ListHousingOptions(ctx, family, viewer)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, option := range options {
		if option.ID == housing {
			found = option.Notes == notes
		}
	}
	if !found {
		t.Fatal("list lost notes")
	}
	if updated, err = repo.UpdateHousingOptionVisibility(ctx, housing, false, owner); err != nil || updated.Notes != notes {
		t.Fatalf("visibility lost notes: %v", err)
	}
	group, err := repo.CreateHousingGroup(ctx, domain.HousingGroup{FamilyID: family, Name: "Notes group"}, owner)
	if err != nil {
		t.Fatal(err)
	}
	if updated, err = repo.AssignHousingOptionGroup(ctx, housing, group.ID, owner); err != nil || updated.Notes != notes {
		t.Fatalf("group assignment lost notes: %v", err)
	}
	options, err = repo.BulkUpdateHousingGroupVisibility(ctx, group.ID, true, owner)
	if err != nil || len(options) != 1 || options[0].Notes != notes {
		t.Fatalf("group visibility lost notes: %v", err)
	}
	limit := strings.Repeat("🏡", domain.HousingNotesMaxLength)
	if saved, err := repo.UpdateHousingNotes(ctx, housing, limit, owner); err != nil || saved != limit {
		t.Fatalf("unicode limit rejected: %v", err)
	}
	if _, err := repo.UpdateHousingNotes(ctx, housing, limit+"a", owner); err == nil {
		t.Fatal("oversized notes accepted")
	}
	if _, err := pool.Exec(ctx, `UPDATE housing_options SET notes=$2 WHERE id=$1`, housing, limit+"a"); err == nil {
		t.Fatal("database length constraint missing")
	}
	if _, err := repo.UpdateHousingNotes(ctx, housing, "", owner); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetHousingOption(ctx, housing, viewer)
	if err != nil || got.Notes != "" {
		t.Fatalf("clear failed: %v", err)
	}
	answersAfter, err := repo.ListHousingAnswers(ctx, family, owner)
	if err != nil || !reflect.DeepEqual(answersBefore, answersAfter) {
		t.Fatalf("notepad changed checklist answers: %v", err)
	}
	newOption := before
	newOption.Notes = notes
	created, err := repo.CreateHousingOption(ctx, newOption, owner)
	if err != nil || created.Notes != notes {
		t.Fatalf("create with notes: %v", err)
	}
	if err := repo.DeleteHousingOption(ctx, created.ID, owner); err != nil {
		t.Fatal(err)
	}
}
