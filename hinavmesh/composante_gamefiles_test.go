//go:build gamefiles

// composante_gamefiles_test.go — composante principale sur un maillage natif de
// l'installation locale (variable LEVELUP_HALO_DEPLOY, voir natif_gamefiles_test.go).
package hinavmesh_test

import (
	"testing"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

func TestComposantePrincipaleBazaar(t *testing.T) {
	dossier := dossierCartes(t)
	m, res, _ := ressourcesCarte(t, dossier, "ctf_bazaar")
	if len(res) == 0 {
		t.Fatal("ctf_bazaar: aucune ressource de maillage")
	}
	brut, err := m.Extract(res[0])
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	mesh, err := hinavmesh.DecodeNatif(brut)
	if err != nil {
		t.Fatalf("DecodeNatif: %v", err)
	}
	c := hinavmesh.ComposantePrincipale(mesh)
	// Mesure : 611 faces sur 670 gardees, 16 composantes ecartees (59 faces, 52,9 m2 sur
	// 1033,6) : de petites faces detachees dans le principal, pas un ilot de jeu.
	if comps := hinavmesh.Composantes(mesh); len(comps) != 17 || comps[0].Aire < 980.66*0.99 || comps[0].Aire > 980.66*1.01 {
		t.Errorf("Composantes: %d, premiere aire %.2f, 17 et 980,66 attendues", len(comps), comps[0].Aire)
	}
	if c.Autres != 16 || c.Faces != 611 {
		t.Errorf("Autres=%d Faces=%d, 16 et 611 attendus", c.Autres, c.Faces)
	}
	if want := 980.66; c.Aire < want*0.99 || c.Aire > want*1.01 {
		t.Errorf("Aire = %.2f m2, %.2f attendus a 1 %% pres", c.Aire, want)
	}
	if c.AireAutres > 0.06*(c.Aire+c.AireAutres) {
		t.Errorf("composantes ecartees: %.2f m2 sur %.2f, 6 %% au plus attendus", c.AireAutres, c.Aire+c.AireAutres)
	}
}
