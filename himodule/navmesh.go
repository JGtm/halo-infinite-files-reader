package himodule

// navmesh.go — REPERAGE DES RESSOURCES DE MAILLAGE DE NAVIGATION d'un module de carte.
//
// Une carte native porte son `hkaiNavMesh` dans une ressource SANS groupe de son `.module`,
// voisine du tag `pfnd`. Le lien entre ce tag et sa ressource n'est PAS decode : on repere
// donc la ressource par son contenu (la chaine `hkaiNavMesh`, nom de la classe racine du
// fichier-tag), parmi les entrees sans groupe d'une taille suffisante.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
)

const (
	// seuilRessourceNavmesh est la taille decompressee minimale d'une ressource candidate.
	// Il n'ecarte que les miettes, pour ne pas extraire inutilement les tres nombreuses
	// petites ressources sans groupe : les maillages secondaires mesures font 11 a 21 Ko et
	// sont retenus, le principal de Bazaar fait 545 828 octets.
	seuilRessourceNavmesh = 10 << 10
	// marqueNavmesh est la chaine du nom de classe presente dans toute ressource de maillage.
	marqueNavmesh = "hkaiNavMesh"
)

// ErrRessourcesIllisibles est rendue, avec la liste des ressources lisibles, quand au moins
// une candidate n'a pas pu etre extraite : le maillage principal pourrait en faire partie,
// l'appelant doit donc le savoir plutot que de recevoir une liste silencieusement amputee.
var ErrRessourcesIllisibles = errors.New("himodule: ressources candidates illisibles")

// RessourcesNavmesh rend les ressources qui portent un maillage de navigation (`hkaiNavMesh`),
// triees par taille decompressee DECROISSANTE : la premiere est le maillage principal de la
// carte, les suivantes (quelques-unes sur chasm, va_launchsite, ridgeline) sont des
// maillages secondaires. Une carte sans maillage (sgh_interlock, academy_tutorial) rend une
// liste vide et aucune erreur.
//
// Chaque candidate (entree sans groupe, au moins seuilRessourceNavmesh octets) est extraite
// pour en lire le contenu ; passer le resultat a Extract puis a hinavmesh.DecodeNatif.
// Une extraction qui echoue est journalisee et comptee : la liste des ressources lisibles
// est rendue avec une erreur enveloppant ErrRessourcesIllisibles.
func (m *Module) RessourcesNavmesh(ctx context.Context) ([]File, error) {
	var trouvees []File
	illisibles := 0
	for _, f := range m.Files("") {
		if f.Group != "" || f.UncompSize < seuilRessourceNavmesh {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		brut, err := m.Extract(f)
		if err != nil {
			illisibles++
			slog.WarnContext(ctx, "himodule: extraction d'une ressource candidate impossible",
				"index", f.Index, "taille", f.UncompSize, "err", err)
			continue
		}
		if !bytes.Contains(brut, []byte(marqueNavmesh)) {
			continue
		}
		slog.DebugContext(ctx, "himodule: ressource de maillage de navigation",
			"index", f.Index, "taille", f.UncompSize, "global_id", f.GlobalID)
		trouvees = append(trouvees, f)
	}
	slices.SortStableFunc(trouvees, func(a, b File) int { return b.UncompSize - a.UncompSize })
	if illisibles > 0 {
		return trouvees, fmt.Errorf("%w: %d extraction(s) echouee(s), %d ressource(s) de maillage lue(s)",
			ErrRessourcesIllisibles, illisibles, len(trouvees))
	}
	return trouvees, nil
}
