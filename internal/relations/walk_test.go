package relations

import "testing"

const nestedFixture = `package main

import "gorm.io/gorm"

type Address struct {
	City string
}

type Profile struct {
	Bio     string
	Address Address
}

type User struct {
	ID      int64
	Profile Profile
}

type Order struct {
	ID   int64
	User User
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("User").Find(&orders)
}
`

// modelFromFixture loads the fixture and resolves the chain's model so walk
// can be exercised in isolation, without going through Verify.
func modelFromFixture(t *testing.T, fixture string) *model {
	t.Helper()
	chains := loadAndCollect(t, map[string]string{"main.go": fixture})
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(chains))
	}
	m := resolveModel(chains[0])
	if m == nil {
		t.Fatal("resolveModel returned nil")
	}
	return m
}

func TestWalk_SingleSegment_OK(t *testing.T) {
	m := modelFromFixture(t, nestedFixture)
	got := m.walk("User")
	if !got.ok {
		t.Fatalf("expected ok=true, got %+v", got)
	}
	if got.failedAt != -1 {
		t.Errorf("expected failedAt=-1 on success, got %d", got.failedAt)
	}
}

func TestWalk_DeepPath_OK(t *testing.T) {
	m := modelFromFixture(t, nestedFixture)
	got := m.walk("User.Profile.Address")
	if !got.ok {
		t.Fatalf("expected ok=true on User.Profile.Address, got %+v", got)
	}
}

func TestWalk_FailsAtFirstSegment_ReportsIndex0(t *testing.T) {
	m := modelFromFixture(t, nestedFixture)
	got := m.walk("Customer")
	if got.ok {
		t.Fatalf("expected ok=false for missing first segment")
	}
	if got.failedAt != 0 {
		t.Errorf("expected failedAt=0, got %d", got.failedAt)
	}
	if got.parent == nil || got.parent.Obj().Name() != "Order" {
		t.Errorf("expected parent=Order, got %v", got.parent)
	}
}

func TestWalk_FailsAtMiddleSegment_ReportsCorrectIndexAndParent(t *testing.T) {
	m := modelFromFixture(t, nestedFixture)
	got := m.walk("User.Profil.Address")
	if got.ok {
		t.Fatal("expected ok=false on typo'd middle segment")
	}
	if got.failedAt != 1 {
		t.Errorf("expected failedAt=1 (the 'Profil' segment), got %d", got.failedAt)
	}
	if got.parent == nil || got.parent.Obj().Name() != "User" {
		t.Errorf("expected parent=User (where 'Profil' was looked up), got %v", got.parent)
	}
}

func TestWalk_FailsWhenSegmentIsScalar(t *testing.T) {
	// Bio is a string, can't recurse into it. "User.Profile.Bio.Anything"
	// must fail at index 2 because Bio resolves but has no struct type.
	m := modelFromFixture(t, nestedFixture)
	got := m.walk("User.Profile.Bio.Anything")
	if got.ok {
		t.Fatal("expected ok=false when descending into a scalar field")
	}
	if got.failedAt != 2 {
		t.Errorf("expected failedAt=2 (the 'Bio' segment), got %d", got.failedAt)
	}
}

func TestWalk_EmbeddedPromotion_OK(t *testing.T) {
	m := modelFromFixture(t, `package main

import "gorm.io/gorm"

type User struct {
	ID   int64
	Name string
}

type BaseModel struct {
	Creator User
}

type Order struct {
	BaseModel
	ID int64
}

func GetOrders(db *gorm.DB) {
	var orders []Order
	db.Preload("Creator").Find(&orders)
}
`)
	got := m.walk("Creator")
	if !got.ok {
		t.Fatalf("expected promoted field 'Creator' to resolve, got %+v", got)
	}
}
