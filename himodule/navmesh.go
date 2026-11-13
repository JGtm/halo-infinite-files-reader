package himodule

// navmesh.go — REPERAGE DES RESSOURCES DE MAILLAGE DE NAVIGATION d'un module de carte.
//
// Une carte native porte son `hkaiNavMesh` dans une ressource SANS groupe de son `.module`,
// voisine du tag `pfnd`. Le lien entre ce tag et sa ressource n'est PAS decode : on repere
// donc la ressource par son contenu (signature `ucsh` en tete et nom de classe
// `hkaiNavMesh`), parmi les entrees sans groupe d'une taille suffisante.

import (
	"bytes"
	"cmp"
	"context"
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
	// signatureRessourceNavmesh ouvre toute ressource de maillage de module.
	signatureRessourceNavmesh = "ucsh"
	// marqueNavmesh est la chaine du nom de classe presente dans toute ressource de maillage.
	// Seule, elle est trop large : la sous-chaine matche aussi hkaiNavMeshQueryMediator et
	// consorts, d'ou l'exigence de la signature en tete.
	marqueNavmesh = "hkaiNavMesh"
)

// RessourcesNavmesh rend les ressources qui portent un maillage de navigation. La liste est
// utilisable meme quand illisibles > 0 : ce compte ne dit que combien de candidates n'ont pas
// pu etre extraites (octets absents du fichier module sur une douzaine de cartes), pas que la
// liste est fausse. err n'est non nil que pour un echec fatal : contexte annule, ou aucune
// entree candidate lisible alors qu'il y en avait.
//
// Les ressources sont triees par taille decompressee DECROISSANTE : la premiere est le
// maillage principal de la carte, les suivantes (quelques-unes sur chasm, va_launchsite,
// ridgeline) sont des maillages secondaires. Une carte sans maillage (sgh_interlock,
// academy_tutorial) rend une liste vide et aucune erreur.
//
// Chaque candidate (entree sans groupe, au moins seuilRessourceNavmesh octets) est extraite
// pour en lire le contenu : elle doit commencer par `ucsh` et contenir `hkaiNavMesh`. Une
// extraction qui echoue est journalisee en Warn et comptee dans illisibles. Passer ensuite
// chaque File a Extract puis a hinavmesh.DecodeNatif.
func (m *Module) RessourcesNavmesh(ctx context.Context) (ressources []File, illisibles int, err error) {
	candidates := 0
	for _, f := range m.Files("") {
		if f.Group != "" || f.UncompSize < seuilRessourceNavmesh {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, illisibles, err
		}
		candidates++
		brut, err := m.Extract(f)
		if err != nil {
			illisibles++
			slog.WarnContext(ctx, "himodule: extraction d'une ressource candidate impossible",
				"index", f.Index, "taille", f.UncompSize, "err", err)
			continue
		}
		if !bytes.HasPrefix(brut, []byte(signatureRessourceNavmesh)) ||
			!bytes.Contains(brut, []byte(marqueNavmesh)) {
			continue
		}
		slog.DebugContext(ctx, "himodule: ressource de maillage de navigation",
			"index", f.Index, "taille", f.UncompSize, "global_id", f.GlobalID)
		ressources = append(ressources, f)
	}
	slices.SortStableFunc(ressources, func(a, b File) int { return cmp.Compare(b.UncompSize, a.UncompSize) })
	if candidates > 0 && illisibles == candidates {
		return nil, illisibles, fmt.Errorf("himodule: les %d ressources candidates sont illisibles", candidates)
	}
	return ressources, illisibles, nil
}
