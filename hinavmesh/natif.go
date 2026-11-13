package hinavmesh

// natif.go — LE MAILLAGE DES CARTES NATIVES, lu dans une ressource de module.
//
// Les cartes natives du studio ne publient pas de `navmesh.blob` : leur maillage est cuit
// dans le `.module` de la carte (`deploy/any/levels/multi/<carte>/<carte>-rtx-new.module`),
// sous forme d'une RESSOURCE sans groupe (`File.Group == ""`) voisine du tag `pfnd`. Le lien
// entre ce tag et sa ressource n'est pas decode ; l'appelant repere la ressource par balayage
// (himodule.Module.RessourcesNavmesh) puis la donne telle quelle a DecodeNatif.
//
// ENVELOPPE (mesure sur ctf_bazaar, ressource de 545 828 octets) : la signature `ucsh` sur
// 4 octets, un en-tete proprietaire dont la zone n'est pas decodee (le premier `TAG0` tombe
// a l'offset 25 096 sur Bazaar), puis quatre fichiers-tag Havok 2022.1 CONTIGUS —
// hkaiClusterGraph, hkaiNavMesh, hkaiTraversalAnnotationLibrary, hkcdStaticAabbTree. Pas de
// conteneur Bond, pas de zlib. Chaque fichier-tag commence par [u32 GROS-BOUTISTE taille
// totale, 2 bits de poids fort = drapeaux][TAG0] : la taille de la section racine suffit a
// le delimiter, on n'a donc pas besoin de comprendre l'en-tete pour trouver le premier.

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// signatureNative ouvre toute ressource de maillage de module.
const signatureNative = "ucsh"

// masqueTailleSection isole la taille totale d'une section : les 2 bits de poids fort du
// u32 gros-boutiste sont des drapeaux.
const masqueTailleSection = 0x3FFFFFFF

// Erreurs de DecodeNatif, exportees pour que l'appelant distingue une ressource qui n'est
// pas un maillage (a ignorer) d'une ressource ambigue (a signaler).
var (
	// ErrSignatureNative : la ressource ne commence pas par `ucsh`.
	ErrSignatureNative = errors.New("hinavmesh: la ressource ne commence pas par la signature ucsh")
	// ErrSansRegion : aucun fichier-tag (signature TAG0) dans la ressource.
	ErrSansRegion = errors.New("hinavmesh: aucune region TAG0 dans la ressource")
	// ErrSansMaillage : des fichiers-tag sont presents mais aucun ne porte un hkaiNavMesh.
	ErrSansMaillage = errors.New("hinavmesh: aucun hkaiNavMesh dans la ressource")
	// ErrMaillagesMultiples : plusieurs hkaiNavMesh dans la meme ressource ; on ne choisit pas.
	ErrMaillagesMultiples = errors.New("hinavmesh: plusieurs hkaiNavMesh dans la ressource")
)

// DecodeNatif lit une ressource de maillage extraite d'un `.module` de carte native et rend
// le maillage de navigation, en coordonnees MONDE.
//
// Les fichiers-tag sont delimites par leur signature TAG0 et la taille de leur section
// racine ; le maillage est cherche par son TYPE (comme Decode), jamais par sa position. Une
// section dont la taille deborde la ressource est une erreur, pas un fichier-tag ignore.
// Plus d'un hkaiNavMesh est refuse (ErrMaillagesMultiples) : le principal d'une carte et ses
// maillages secondaires vivent dans des ressources distinctes.
func DecodeNatif(ressource []byte) (*Maillage, error) {
	if len(ressource) < len(signatureNative) || string(ressource[:len(signatureNative)]) != signatureNative {
		return nil, ErrSignatureNative
	}
	regionsTag, err := decoupeFichiersTag(ressource)
	if err != nil {
		return nil, err
	}
	if len(regionsTag) == 0 {
		return nil, ErrSansRegion
	}
	type trouve struct {
		f  *fichierTag
		it itemHavok
	}
	var tous []trouve
	for i, region := range regionsTag {
		f, vus, err := maillagesDeRegion(region)
		if err != nil {
			return nil, fmt.Errorf("hinavmesh: fichier-tag %d de la ressource: %w", i+1, err)
		}
		for _, it := range vus {
			tous = append(tous, trouve{f: f, it: it})
		}
	}
	switch len(tous) {
	case 0:
		return nil, fmt.Errorf("%w (%d fichiers-tag lus)", ErrSansMaillage, len(regionsTag))
	case 1:
		return construitMaillage(tous[0].f, tous[0].it)
	default:
		return nil, fmt.Errorf("%w (%d trouves)", ErrMaillagesMultiples, len(tous))
	}
}

// decoupeFichiersTag rend les fichiers-tag d'une ressource, dans l'ordre. Le balayage
// avance octet par octet jusqu'au premier TAG0 (la zone d'en-tete n'est pas decodee), puis
// saute de section en section : on ne cherche plus de signature a l'interieur d'un fichier.
func decoupeFichiersTag(ressource []byte) ([][]byte, error) {
	var out [][]byte
	for off := len(signatureNative); off+8 <= len(ressource); {
		if string(ressource[off+4:off+8]) != sectionTAG0 {
			off++
			continue
		}
		taille := int(binary.BigEndian.Uint32(ressource[off:]) & masqueTailleSection)
		if taille < 8 || off+taille > len(ressource) {
			return nil, fmt.Errorf("hinavmesh: section TAG0 a l'offset %d: taille %d invalide ou debordant la ressource (%d octets)",
				off, taille, len(ressource))
		}
		out = append(out, ressource[off:off+taille])
		off += taille
	}
	return out, nil
}
