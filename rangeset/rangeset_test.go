package rangeset_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/brentp/go-athenaeum/rangeset"
)

func TestRange(t *testing.T) {

	b, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}

	b.SetRange(200, 400)

	ivs := b.Ranges()

	if !reflect.DeepEqual(ivs, []rangeset.Range{rangeset.Range{Start: 200, End: 400}}) {
		t.Fatalf("got: %s", ivs)
	}

}

func TestIntersection(t *testing.T) {
	a, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}
	b, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}

	a.SetRange(200, 400)
	b.SetRange(400, 800)

	c := rangeset.Intersection(a, b)
	if err != nil {
		t.Fatal("error in intersection")
	}

	ivs := c.Ranges()

	if len(ivs) != 0 {
		t.Fatalf("expected no intersection, got: %s", ivs)
	}

	b.SetRange(399, 800)
	c = rangeset.Intersection(a, b)

	ivs = c.Ranges()
	if len(ivs) != 1 {
		t.Fatalf("expected 1 intersection, got: %s", ivs)
	}
	if ivs[0].Start != 399 || ivs[0].End != 400 {
		t.Fatalf("expected 399-800, got: %s", ivs[0])
	}
}

func TestUnion(t *testing.T) {
	a, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}
	b, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}

	a.SetRange(200, 399)
	b.SetRange(400, 800)

	c := rangeset.Union(a, b)
	if err != nil {
		t.Fatal("error in union")
	}

	ivs := c.Ranges()

	if len(ivs) != 2 {
		t.Fatalf("expected 2 items in union, got: %s", ivs)
	}
	if ivs[0].Start != 200 || ivs[0].End != 399 || ivs[1].Start != 400 || ivs[1].End != 800 {
		t.Fatalf("unexpected items in union, got: %s", ivs)
	}

	b.SetRange(399, 800)
	c = rangeset.Union(a, b)

	ivs = c.Ranges()
	if len(ivs) != 1 {
		t.Fatalf("expected 1 union, got: %s", ivs)
	}
	if ivs[0].Start != 200 || ivs[0].End != 800 {
		t.Fatalf("expected 200-800, got: %s", ivs[0])
	}
}

func TestDifference(t *testing.T) {
	a, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}
	b, err := rangeset.New(0, 1000)
	if err != nil {
		t.Fatal("error in New")
	}

	a.SetRange(200, 600)
	b.SetRange(400, 1000)

	ivs := rangeset.Difference(a, b).Ranges()
	if len(ivs) != 1 || ivs[0].Start != 200 || ivs[0].End != 400 {
		t.Fatalf("expected 1 interval 200-400, got: %s", ivs)
	}

	ivs = rangeset.Difference(b, a).Ranges()
	if len(ivs) != 1 || ivs[0].Start != 600 || ivs[0].End != 1000 {
		t.Fatalf("expected 1 interval 200-400, got: %s", ivs)
	}
}

func Example() {

	chromLength := 1000
	genes, _ := rangeset.New(0, chromLength)
	genes.SetRange(100, 200)
	genes.SetRange(250, 350)
	genes.SetRange(900, 950)

	regions, _ := rangeset.New(0, chromLength)
	regions.SetRange(150, 250)
	regions.SetRange(500, 925)

	fmt.Println("genes          :", genes.Ranges())
	fmt.Println("regions        :", regions.Ranges())
	fmt.Println("genes ∩ regions:", rangeset.Intersection(genes, regions).Ranges())
	fmt.Println("genes ∪ regions:", rangeset.Union(genes, regions).Ranges())
	fmt.Println("genes - regions:", rangeset.Difference(genes, regions).Ranges())
	fmt.Println("regions - genes:", rangeset.Difference(regions, genes).Ranges())

	// Output:
	// genes          : [Range(100-200) Range(250-350) Range(900-950)]
	// regions        : [Range(150-250) Range(500-925)]
	// genes ∩ regions: [Range(150-200) Range(900-925)]
	// genes ∪ regions: [Range(100-350) Range(500-950)]
	// genes - regions: [Range(100-150) Range(250-350) Range(925-950)]
	// regions - genes: [Range(200-250) Range(500-900)]

}
