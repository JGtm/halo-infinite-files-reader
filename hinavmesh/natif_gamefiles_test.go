//go:build gamefiles

// natif_gamefiles_test.go — DecodeNatif sur les modules de l'installation locale de Halo
// Infinite. Lecture seule ; Skip propre si le jeu n'est pas installe.
//
//	go test -tags gamefiles -count=1 ./hinavmesh/ -run Natif -v
//
// Racine du jeu : variable d'environnement LEVELUP_HALO_DEPLOY (le dossier `deploy` de
// l'installation), sans valeur par defaut : variable vide, test saute.
package hinavmesh_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/JGtm/halo-infinite-files-reader/himodule"
	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// dossierCartes rend le dossier des cartes multijoueur du module `any`, ou saute le test.
func dossierCartes(t *testing.T) string {
	t.Helper()
	racine := os.Getenv("LEVELUP_HALO_DEPLOY")
	if racine == "" {
		t.Skip("LEVELUP_HALO_DEPLOY non posee : la definir sur le dossier `deploy` de Halo Infinite")
	}
	dossier := filepath.Join(racine, "any", "levels", "multi")
	if _, err := os.Stat(dossier); err != nil {
		t.Skipf("Halo Infinite introuvable sous %s: %v", racine, err)
	}
	return dossier
}

// ressourcesCarte ouvre le module d'une carte et rend ses ressources de maillage, la plus
// grosse d'abord, et le nombre de candidates non extraites. Le module est fermé en fin de test.
func ressourcesCarte(t *testing.T, dossier, carte string) (*himodule.Module, []himodule.File, int) {
	t.Helper()
	m, err := himodule.Open(filepath.Join(dossier, carte, carte+"-rtx-new.module"))
	if err != nil {
		t.Fatalf("%s: Open: %v", carte, err)
	}
	t.Cleanup(func() { _ = m.Close() })
	res, illisibles, err := m.RessourcesNavmesh(context.Background())
	if err != nil {
		t.Fatalf("%s: RessourcesNavmesh: %v", carte, err)
	}
	return m, res, illisibles
}

func proche(a, b float64) bool { return math.Abs(a-b) <= 0.1 }

func TestNatifBazaar(t *testing.T) {
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
	if len(mesh.Sommets) != 2312 || len(mesh.Faces) != 670 {
		t.Errorf("%d sommets %d faces, 2312 et 670 attendus", len(mesh.Sommets), len(mesh.Faces))
	}
	bornes := []struct {
		nom       string
		got, veut float64
	}{
		{"Min.X", mesh.Min.X, -25.7}, {"Max.X", mesh.Max.X, 25.7},
		{"Min.Y", mesh.Min.Y, -15.2}, {"Max.Y", mesh.Max.Y, 12.7},
		{"Min.Z", mesh.Min.Z, -2.2}, {"Max.Z", mesh.Max.Z, 3.2},
	}
	for _, b := range bornes {
		if !proche(b.got, b.veut) {
			t.Errorf("%s = %.2f, %.1f attendu a 0,1 m pres", b.nom, b.got, b.veut)
		}
	}
	if mesh.HautSuppose {
		t.Error("HautSuppose: le maillage natif declare `up`")
	}
	if mesh.Haut != (hinavmesh.Point{X: 0, Y: 0, Z: 1}) {
		t.Errorf("Haut = %v, (0,0,1) attendu", mesh.Haut)
	}
}

func TestNatifBalayageCartes(t *testing.T) {
	dossier := dossierCartes(t)
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Fatal(err)
	}
	var cartes []string
	for _, e := range entrees {
		if e.IsDir() {
			cartes = append(cartes, e.Name())
		}
	}
	sort.Strings(cartes)
	if len(cartes) != 31 {
		t.Fatalf("%d cartes sous %s, 31 attendues", len(cartes), dossier)
	}
	secondairesAttendus := map[string]int{"chasm": 5, "va_launchsite": 5, "ridgeline": 1}
	// Candidates non extraites par carte : leurs octets sont absents du fichier module (erreur
	// `bloc N hors borne`), 17 au total sur 12 cartes. Toute carte absente de la table en a 0.
	illisiblesAttendus := map[string]int{
		"academy_weapon_drills": 2, "btb_engine": 1, "chasm": 1, "ctf_breaker": 4,
		"ctf_forbidden": 1, "ctf_illusion": 1, "forest": 1, "pve_house": 1,
		"sgh_blueprint": 1, "sgh_crystalcaves": 1, "sgh_streets": 2, "va_behemoth": 1,
	}
	var sans []string
	for _, carte := range cartes {
		m, res, illisibles := ressourcesCarte(t, dossier, carte)
		if illisibles != illisiblesAttendus[carte] {
			t.Errorf("%s: %d candidates illisibles, %d attendues", carte, illisibles, illisiblesAttendus[carte])
		}
		if len(res) == 0 {
			sans = append(sans, carte)
			continue
		}
		if got := len(res) - 1; got != secondairesAttendus[carte] {
			t.Errorf("%s: %d maillages secondaires, %d attendus", carte, got, secondairesAttendus[carte])
		}
		brut, err := m.Extract(res[0])
		if err != nil {
			t.Errorf("%s: Extract: %v", carte, err)
			continue
		}
		mesh, err := hinavmesh.DecodeNatif(brut)
		if err != nil {
			t.Errorf("%s: DecodeNatif du principal: %v", carte, err)
			continue
		}
		if len(mesh.Faces) == 0 {
			t.Errorf("%s: maillage vide", carte)
		}
	}
	sort.Strings(sans)
	if len(sans) != 2 || sans[0] != "academy_tutorial" || sans[1] != "sgh_interlock" {
		t.Errorf("cartes sans maillage %v, [academy_tutorial sgh_interlock] attendues", sans)
	}
	if got := len(cartes) - len(sans); got != 29 {
		t.Errorf("%d cartes avec maillage, 29 attendues", got)
	}
}
