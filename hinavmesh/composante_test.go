package hinavmesh

import "testing"

// deuxComposantes : un carre de deux faces (aire 4) et un triangle isole (aire 0,5).
func deuxComposantes() *Maillage {
	return &Maillage{
		Sommets: []Point{
			{0, 0, 0}, {2, 0, 0}, {2, 2, 0}, {0, 2, 0}, // carre
			{10, 10, 0}, {11, 10, 0}, {10, 11, 0}, // triangle isole
		},
		Faces: []Face{
			{Sommets: []int32{4, 5, 6}},
			{Sommets: []int32{0, 1, 2}},
			{Sommets: []int32{0, 2, 3}},
		},
	}
}

func TestComposantePrincipaleCarreGagneParAire(t *testing.T) {
	c := ComposantePrincipale(deuxComposantes())
	if c.Faces != 2 || c.Autres != 1 {
		t.Fatalf("Faces=%d Autres=%d, 2 et 1 attendus", c.Faces, c.Autres)
	}
	if want := []bool{false, true, true}; len(c.Garde) != 3 || c.Garde[0] != want[0] || c.Garde[1] != want[1] || c.Garde[2] != want[2] {
		t.Errorf("Garde = %v, %v attendu", c.Garde, want)
	}
	if c.Aire != 4 || c.AireAutres != 0.5 {
		t.Errorf("Aire=%v AireAutres=%v, 4 et 0,5 attendus", c.Aire, c.AireAutres)
	}
}

func TestComposantePrincipaleUnSommetPartageSuffit(t *testing.T) {
	m := deuxComposantes()
	// Le triangle isole devient voisin du carre par le seul sommet 2.
	m.Faces[0] = Face{Sommets: []int32{2, 5, 6}}
	c := ComposantePrincipale(m)
	if c.Autres != 0 || c.Faces != 3 {
		t.Errorf("Faces=%d Autres=%d, 3 et 0 attendus", c.Faces, c.Autres)
	}
}

func TestComposantePrincipaleAireEtNonNombreDeFaces(t *testing.T) {
	// Trois petits triangles relies (aire 1,5) contre un grand triangle isole (aire 50).
	m := &Maillage{
		Sommets: []Point{
			{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0}, {2, 0, 0},
			{100, 0, 0}, {110, 0, 0}, {100, 10, 0},
		},
		Faces: []Face{
			{Sommets: []int32{0, 1, 2}}, {Sommets: []int32{1, 3, 2}}, {Sommets: []int32{1, 4, 3}},
			{Sommets: []int32{5, 6, 7}},
		},
	}
	c := ComposantePrincipale(m)
	if c.Faces != 1 || c.Autres != 1 || !c.Garde[3] || c.Aire != 50 {
		t.Errorf("Faces=%d Autres=%d Aire=%v Garde=%v, le grand triangle devait gagner", c.Faces, c.Autres, c.Aire, c.Garde)
	}
}

func TestComposantePrincipaleVide(t *testing.T) {
	for _, m := range []*Maillage{{}, nil} {
		c := ComposantePrincipale(m)
		if len(c.Garde) != 0 || c.Faces != 0 || c.Autres != 0 {
			t.Errorf("maillage vide: %+v", c)
		}
	}
}

func TestComposantePrincipaleDeterministe(t *testing.T) {
	// Deux composantes d'aires egales : celle de la plus petite face d'indice gagne, a chaque tirage.
	m := &Maillage{
		Sommets: []Point{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {5, 5, 0}, {6, 5, 0}, {5, 6, 0}},
		Faces:   []Face{{Sommets: []int32{3, 4, 5}}, {Sommets: []int32{0, 1, 2}}},
	}
	for i := 0; i < 50; i++ {
		if c := ComposantePrincipale(m); !c.Garde[0] || c.Garde[1] {
			t.Fatalf("tirage %d: Garde = %v, [true false] attendu", i, c.Garde)
		}
	}
}

func TestTrianglesDeRestreint(t *testing.T) {
	m := deuxComposantes()
	m.Faces = append(m.Faces, Face{Sommets: []int32{0, 1, 2, 3}}) // quadrilatere : 2 triangles
	garde := []bool{false, true, false, true}
	tris := m.TrianglesDe(garde)
	if len(tris) != 3 {
		t.Fatalf("%d triangles, 3 attendus", len(tris))
	}
	if tris[0] != [3]Point{m.Sommets[0], m.Sommets[1], m.Sommets[2]} {
		t.Errorf("premier triangle %v", tris[0])
	}
	// Tout garde = Triangles().
	tout := m.TrianglesDe([]bool{true, true, true, true})
	ref := m.Triangles()
	if len(tout) != len(ref) {
		t.Fatalf("%d triangles, %d attendus", len(tout), len(ref))
	}
	for i := range ref {
		if tout[i] != ref[i] {
			t.Errorf("triangle %d differe de Triangles()", i)
		}
	}
	if len(m.TrianglesDe(nil)) != 0 {
		t.Error("garde nil devait ne rien garder")
	}
}

// empriseGardee rend l'emprise XY des faces gardees.
func empriseGardee(m *Maillage, garde []bool) (min, max Point) {
	premier := true
	for i, f := range m.Faces {
		if !garde[i] {
			continue
		}
		for _, s := range f.Sommets {
			p := m.Sommets[s]
			if premier {
				min, max, premier = p, p, false
				continue
			}
			min = Point{X: minF(min.X, p.X), Y: minF(min.Y, p.Y)}
			max = Point{X: maxF(max.X, p.X), Y: maxF(max.Y, p.Y)}
		}
	}
	return min, max
}

func minF(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

func maxF(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

// Isolation (blob Forge) : la grande bande du canevas en bas du maillage est ecartee.
// Mesure : 2240 faces sur 2348 gardees, 16 composantes ecartees pour 772,6 m2.
func TestComposantePrincipaleIsolation(t *testing.T) {
	m := decodeTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	c := ComposantePrincipale(m)
	if comps := Composantes(m); len(comps) != 17 || comps[0].Aire < 1427.04*0.99 || comps[0].Aire > 1427.04*1.01 {
		t.Errorf("Composantes: %d, premiere aire %.2f, 17 et 1427 attendues", len(comps), comps[0].Aire)
	}
	if c.Autres < 1 {
		t.Fatalf("Autres = %d, au moins 1 attendu", c.Autres)
	}
	if c.Autres != 16 || c.Faces != 2240 {
		t.Errorf("Autres=%d Faces=%d, 16 et 2240 attendus", c.Autres, c.Faces)
	}
	if want := 1427.04; c.Aire < want*0.99 || c.Aire > want*1.01 {
		t.Errorf("Aire = %.2f m2, %.2f attendus a 1 %% pres", c.Aire, want)
	}
	mn, mx := empriseGardee(m, c.Garde)
	if mn.X < -50.5 || mx.X > 0 {
		t.Errorf("emprise X [%.2f %.2f], dans [-50.5 0] attendu", mn.X, mx.X)
	}
	if mn.Y < -38.5 || mx.Y > -5 {
		t.Errorf("emprise Y [%.2f %.2f], dans [-38.5 -5] attendu", mn.Y, mx.Y)
	}
}

func TestComposantesOrdreEmpriseEtSomme(t *testing.T) {
	m := deuxComposantes()
	comps := Composantes(m)
	if len(comps) != 2 {
		t.Fatalf("%d composantes, 2 attendues", len(comps))
	}
	carre, triangle := comps[0], comps[1]
	if carre.Faces != 2 || carre.Aire != 4 || triangle.Faces != 1 || triangle.Aire != 0.5 {
		t.Errorf("carre %+v, triangle %+v : ordre ou aires faux", carre, triangle)
	}
	if carre.Min != (Point{0, 0, 0}) || carre.Max != (Point{2, 2, 0}) {
		t.Errorf("emprise du carre %v..%v", carre.Min, carre.Max)
	}
	if triangle.Min != (Point{10, 10, 0}) || triangle.Max != (Point{11, 11, 0}) {
		t.Errorf("emprise du triangle %v..%v", triangle.Min, triangle.Max)
	}
	if somme := carre.Aire + triangle.Aire; somme != m.AireAuSol() {
		t.Errorf("somme des aires %v, aire totale %v", somme, m.AireAuSol())
	}
	if carre.Autres != 0 || carre.AireAutres != 0 {
		t.Errorf("Autres/AireAutres reserves a ComposantePrincipale : %+v", carre)
	}
	// Chaque face appartient a exactement une composante.
	for i := range m.Faces {
		n := 0
		for _, c := range comps {
			if c.Garde[i] {
				n++
			}
		}
		if n != 1 {
			t.Errorf("face %d dans %d composantes", i, n)
		}
	}
	if Composantes(&Maillage{}) != nil || Composantes(nil) != nil {
		t.Error("maillage vide : nil attendu")
	}
}

func TestComposantesEmpriseZ(t *testing.T) {
	m := &Maillage{
		Sommets: []Point{{0, 0, -1}, {1, 0, 2}, {0, 1, 0.5}},
		Faces:   []Face{{Sommets: []int32{0, 1, 2}}},
	}
	c := Composantes(m)[0]
	if c.Min.Z != -1 || c.Max.Z != 2 {
		t.Errorf("Z %v..%v, -1..2 attendu", c.Min.Z, c.Max.Z)
	}
}

func TestComposantesDeterministeAiresEgales(t *testing.T) {
	m := &Maillage{
		Sommets: []Point{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {5, 5, 0}, {6, 5, 0}, {5, 6, 0}},
		Faces:   []Face{{Sommets: []int32{3, 4, 5}}, {Sommets: []int32{0, 1, 2}}},
	}
	for i := 0; i < 50; i++ {
		comps := Composantes(m)
		if len(comps) != 2 || !comps[0].Garde[0] || !comps[1].Garde[1] {
			t.Fatalf("tirage %d: ordre instable", i)
		}
	}
}
