package hinavmesh

// composante.go — LA COMPOSANTE PRINCIPALE du maillage.
//
// POURQUOI. Un maillage Forge (blob) porte aussi des dalles du CANEVAS Forge, detachees de
// l'arene : sur Isolation, une grande bande en bas du maillage qui n'est pas du terrain
// jouable. Les maillages natifs ont parfois des ilots secondaires (Chasm en annonce cinq) et
// des faces detachees dans le principal. Rasteriser tout le maillage etale ces dalles dans
// l'emprise du fond de carte. En ne gardant que la plus grande composante connexe, les dalles
// disparaissent et l'arene reste entiere.
//
// CONNEXITE. Deux faces sont reliees des qu'elles partagent UN sommet (indice commun dans
// Maillage.Sommets), pas necessairement une arete : un seul sommet partage suffit. La
// composante principale est celle d'aire cumulee MAXIMALE (aire au sol, lacet sur le contour
// XY), pas celle qui compte le plus de faces. A aires egales, la composante qui contient la
// plus petite face d'indice l'emporte : le resultat est deterministe.
//
// CE QUE CA COUTE. Une arene faite de deux ilots relies par un pont ABSENT du maillage
// perdrait le second ilot. Ce n'est jamais silencieux : les composantes ecartees sont
// comptees dans Composante.Autres et leur aire dans Composante.AireAutres, a l'appelant de
// juger si c'est acceptable pour la carte.

import "math"

// Composante est le resultat de ComposantePrincipale.
type Composante struct {
	// Garde dit, par indice de face de Maillage.Faces, si la face est dans la composante
	// principale. Vide pour un maillage vide.
	Garde []bool
	// Faces est le nombre de faces gardees.
	Faces int
	// Autres est le nombre de composantes ecartees.
	Autres int
	// Aire est l'aire au sol de la composante principale, en m2.
	Aire float64
	// AireAutres est l'aire au sol cumulee des composantes ecartees, en m2.
	AireAutres float64
}

// ComposantePrincipale rend la plus grande composante connexe du maillage, par aire.
func ComposantePrincipale(m *Maillage) Composante {
	if m == nil || len(m.Faces) == 0 {
		return Composante{Garde: []bool{}}
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
	// Aire cumulee et premiere face par racine de composante.
	aires := make(map[int32]float64)
	premiere := make(map[int32]int)
	racines := make([]int32, len(m.Faces))
	for i, f := range m.Faces {
		r := racine(parent, f.Sommets[0])
		racines[i] = r
		aires[r] += aireFace(m, f)
		if _, vu := premiere[r]; !vu {
			premiere[r] = i
		}
	}
	var gagnante int32
	meilleure, trouve := 0.0, false
	for r, a := range aires {
		if !trouve || a > meilleure || (a == meilleure && premiere[r] < premiere[gagnante]) {
			gagnante, meilleure, trouve = r, a, true
		}
	}
	c := Composante{Garde: make([]bool, len(m.Faces)), Autres: len(aires) - 1, Aire: meilleure}
	for i, r := range racines {
		if r == gagnante {
			c.Garde[i] = true
			c.Faces++
		}
	}
	for r, a := range aires {
		if r != gagnante {
			c.AireAutres += a
		}
	}
	return c
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
