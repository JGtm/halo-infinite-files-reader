package hinavmesh

// composante.go — LES COMPOSANTES CONNEXES du maillage.
//
// POURQUOI. Un maillage Forge (blob) porte aussi des dalles du CANEVAS Forge, detachees de
// l'arene : sur Isolation, une grande bande en bas du maillage qui n'est pas du terrain
// jouable. Les maillages natifs ont parfois des ilots secondaires (Chasm en annonce cinq) et
// des faces detachees dans le principal. Rasteriser tout le maillage etale ces dalles dans
// l'emprise du fond de carte. Les composantes sont donc exposees TOUTES, avec leur aire et
// leur emprise : choisir celles qui forment l'arene demande des ancres d'objectif et la
// matiere du rendu, que ce paquet ne connait pas. L'aire seule ne suffit pas : sur des cartes
// Forge, la plus grande composante est le sol du canevas, pas l'arene.
//
// CONNEXITE. Deux faces sont reliees des qu'elles partagent UN sommet (indice commun dans
// Maillage.Sommets), pas necessairement une arete : un seul sommet partage suffit. Les
// composantes sont triees par aire au sol decroissante (lacet sur le contour XY) ; a aires
// egales, celle qui contient la plus petite face d'indice passe d'abord : le resultat est
// deterministe.
//
// ComposantePrincipale est le raccourci « la plus grande par aire ». CE QUE CA COUTE : une
// arene faite de deux ilots relies par un pont ABSENT du maillage perdrait le second ilot ;
// ce n'est jamais silencieux, les ecartees sont comptees dans Autres et AireAutres.

import (
	"math"
	"sort"
)

// Composante est un ensemble de faces reliees par des sommets partages.
type Composante struct {
	// Garde dit, par indice de face de Maillage.Faces, si la face appartient a la
	// composante. Vide pour un maillage vide.
	Garde []bool
	// Faces est le nombre de faces de la composante.
	Faces int
	// Aire est l'aire au sol de la composante, en m2.
	Aire float64
	// Min et Max sont l'emprise (X, Y et Z) des sommets des faces de la composante.
	Min, Max Point
	// Autres et AireAutres ne sont renseignes que par ComposantePrincipale : nombre et
	// aire au sol cumulee des composantes ecartees. Nuls pour les elements de Composantes.
	Autres     int
	AireAutres float64
}

// Composantes rend toutes les composantes connexes du maillage, triees par aire au sol
// decroissante (a aires egales, plus petit indice de face d'abord). Maillage vide : nil.
func Composantes(m *Maillage) []Composante {
	if m == nil || len(m.Faces) == 0 {
		return nil
	}
	parent := make([]int32, len(m.Sommets))
	for i := range parent {
		parent[i] = int32(i)
	}
	for _, f := range m.Faces {
		for _, s := range f.Sommets[1:] {
			unis(parent, f.Sommets[0], s)
		}
	}
	index := make(map[int32]int) // racine -> position dans comps
	var comps []Composante
	var premiere []int // premiere face de chaque composante, pour departager
	for i, f := range m.Faces {
		r := racine(parent, f.Sommets[0])
		k, vu := index[r]
		if !vu {
			k = len(comps)
			index[r] = k
			comps = append(comps, Composante{Garde: make([]bool, len(m.Faces)), Min: pointInfini(1), Max: pointInfini(-1)})
			premiere = append(premiere, i)
		}
		c := &comps[k]
		c.Garde[i] = true
		c.Faces++
		c.Aire += aireFace(m, f)
		for _, s := range f.Sommets {
			elargit(c, m.Sommets[s])
		}
	}
	ordre := make([]int, len(comps))
	for i := range ordre {
		ordre[i] = i
	}
	sort.SliceStable(ordre, func(a, b int) bool {
		ca, cb := comps[ordre[a]], comps[ordre[b]]
		if ca.Aire != cb.Aire {
			return ca.Aire > cb.Aire
		}
		return premiere[ordre[a]] < premiere[ordre[b]]
	})
	tri := make([]Composante, len(comps))
	for i, k := range ordre {
		tri[i] = comps[k]
	}
	return tri
}

// ComposantePrincipale rend la plus grande composante par aire (le premier element de
// Composantes), avec Autres et AireAutres pour les autres. Maillage vide : Garde vide.
func ComposantePrincipale(m *Maillage) Composante {
	comps := Composantes(m)
	if len(comps) == 0 {
		return Composante{Garde: []bool{}}
	}
	c := comps[0]
	c.Autres = len(comps) - 1
	for _, autre := range comps[1:] {
		c.AireAutres += autre.Aire
	}
	return c
}

// pointInfini rend un point dont les trois coordonnees valent signe * +Inf.
func pointInfini(signe float64) Point {
	v := signe * math.Inf(1)
	return Point{X: v, Y: v, Z: v}
}

// elargit etend l'emprise de la composante au point p.
func elargit(c *Composante, p Point) {
	c.Min = Point{X: math.Min(c.Min.X, p.X), Y: math.Min(c.Min.Y, p.Y), Z: math.Min(c.Min.Z, p.Z)}
	c.Max = Point{X: math.Max(c.Max.X, p.X), Y: math.Max(c.Max.Y, p.Y), Z: math.Max(c.Max.Z, p.Z)}
}

// TrianglesDe rend la soupe de triangles des seules faces gardees, avec la meme decoupe
// en eventail que Triangles. Une face dont l'indice depasse garde est ecartee.
func (m *Maillage) TrianglesDe(garde []bool) [][3]Point {
	tris := make([][3]Point, 0, len(m.Faces))
	for i, f := range m.Faces {
		if i >= len(garde) || !garde[i] {
			continue
		}
		for k := 1; k+1 < len(f.Sommets); k++ {
			tris = append(tris, [3]Point{
				m.Sommets[f.Sommets[0]], m.Sommets[f.Sommets[k]], m.Sommets[f.Sommets[k+1]],
			})
		}
	}
	return tris
}

// aireFace rend l'aire au sol d'une face (lacet sur le contour projete en XY).
func aireFace(m *Maillage, f Face) float64 {
	var s float64
	n := len(f.Sommets)
	for k := 0; k < n; k++ {
		a, b := m.Sommets[f.Sommets[k]], m.Sommets[f.Sommets[(k+1)%n]]
		s += a.X*b.Y - b.X*a.Y
	}
	return math.Abs(s) / 2
}

// racine rend la racine de l'ensemble de x, avec compression de chemin.
func racine(parent []int32, x int32) int32 {
	for parent[x] != x {
		parent[x] = parent[parent[x]]
		x = parent[x]
	}
	return x
}

// unis fusionne les ensembles de a et b ; la plus petite racine l'emporte, ce qui rend
// la structure independante de l'ordre des appels.
func unis(parent []int32, a, b int32) {
	ra, rb := racine(parent, a), racine(parent, b)
	switch {
	case ra < rb:
		parent[rb] = ra
	case rb < ra:
		parent[ra] = rb
	}
}
