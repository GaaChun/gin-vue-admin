package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/shopspring/decimal"
)

func TestMaterialUnitSetValidation(t *testing.T) {
	valid := []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}, {UnitID: 2, Factor: "12", Enabled: true}}
	if _, err := prepareMaterialUnits(valid); err != nil { t.Fatalf("valid unit set rejected: %v", err) }
	cases := []struct {
		name string
		units []req.MaterialUnitInput
	}{
		{"zero bases", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", Enabled: true}}},
		{"multiple bases", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}, {UnitID: 2, Factor: "1", IsBase: true, Enabled: true}}},
		{"disabled base", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: false}}},
		{"base factor not one", []req.MaterialUnitInput{{UnitID: 1, Factor: "2", IsBase: true, Enabled: true}}},
		{"duplicate unit", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}, {UnitID: 1, Factor: "2", Enabled: true}}},
		{"factor precision", []req.MaterialUnitInput{{UnitID: 1, Factor: "1.0000001", IsBase: true, Enabled: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := prepareMaterialUnits(tc.units); err == nil { t.Fatal("invalid unit set accepted") }
		})
	}
}

func TestMaterialCodeConflictButNameMayRepeat(t *testing.T) {
	if err := ValidateIdentifierAvailability(ReusableMasterIdentifier, InspectIdentifierPresence("M-1", []IdentifierRecord{{ID: 1, Value: "m-1"}}), "code"); err == nil {
		t.Fatal("active material code conflict accepted")
	}
	// Material names intentionally have no active-unique validation.
	if _, _, err := prepareMaterialInput(req.MaterialInput{Name: "同名物料", Mnemonic: "tmwl", Code: "M-1", BrandID: 1, CategoryID: 1, Units: []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}}}); err != nil {
		t.Fatalf("a material name itself must not be rejected: %v", err)
	}
}

func TestMaterialCustomerAndCategoryAreOptional(t *testing.T) {
	input := req.MaterialInput{Name: "物料", Mnemonic: "wl", Code: "M-2", Units: []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}}}
	material, _, err := prepareMaterialInput(input)
	if err != nil {
		t.Fatalf("material without customer/category rejected: %v", err)
	}
	if material.BrandID != 0 || material.CategoryID != 0 {
		t.Fatalf("optional references were not empty: %#v", material)
	}
}

func TestMaterialBindingReplacementPreservesMatchingID(t *testing.T) {
	existing := model.MaterialUnit{MaterialID: 4, UnitID: 1}
	existing.ID = 7
	bindings := materialBindings(4, 2, []materialUnitCommand{{unitID: 1, factor: mustFactor(t, "2"), enabled: true}, {unitID: 3, factor: mustFactor(t, "1"), isBase: true, enabled: true}}, map[uint]model.MaterialUnit{1: existing})
	if bindings[0].ID != 7 { t.Fatalf("existing binding ID = %d, want 7", bindings[0].ID) }
	if bindings[1].ID != 0 { t.Fatalf("new binding unexpectedly retained ID %d", bindings[1].ID) }
}

func TestMaterialUnitSetAllowsSwitchingBaseAndDisablingFormerBase(t *testing.T) {
	units, err := prepareMaterialUnits([]req.MaterialUnitInput{
		{UnitID: 1, Factor: "0.083333", Enabled: false},
		{UnitID: 2, Factor: "1", IsBase: true, Enabled: true},
	})
	if err != nil {
		t.Fatalf("valid replacement base rejected: %v", err)
	}
	if units[0].isBase || units[0].enabled || !units[1].isBase || !units[1].factor.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("replacement unit set = %#v", units)
	}
}

func mustFactor(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	factor, err := ParseDecimal(value, FactorRule, "factor")
	if err != nil { t.Fatal(err) }
	return factor
}

func TestStrictMaterialImageValidation(t *testing.T) {
	jpeg := encodedImage(t, "jpeg")
	png := encodedImage(t, "png")
	bmp := validBMP(26)
	for name, contents := range map[string][]byte{"image.jpg": jpeg, "image.png": png, "image.bmp": bmp} {
		if _, err := ValidateMaterialImage(name, contents); err != nil { t.Fatalf("%s rejected: %v", name, err) }
	}
	boundary := validBMP(MaterialImageMaxBytes)
	if _, err := ValidateMaterialImage("boundary.bmp", boundary); err != nil { t.Fatalf("exactly 1MiB rejected: %v", err) }
	if _, err := ValidateMaterialImage("over.bmp", append(boundary, 0)); err == nil { t.Fatal("1MiB plus one byte accepted") }
	if _, err := ValidateMaterialImage("masquerade.jpg", bmp); err == nil { t.Fatal("extension masquerade accepted") }
}

func encodedImage(t *testing.T, format string) []byte {
	t.Helper()
	var out bytes.Buffer
	pixel := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pixel.Set(0, 0, color.White)
	var err error
	if format == "jpeg" { err = jpeg.Encode(&out, pixel, nil) } else { err = png.Encode(&out, pixel) }
	if err != nil { t.Fatal(err) }
	return out.Bytes()
}

func validBMP(size int) []byte {
	contents := make([]byte, size)
	contents[0], contents[1] = 'B', 'M'
	contents[14] = 40 // BITMAPINFOHEADER
	contents[18], contents[22] = 1, 1
	return contents
}
