package hinavmesh

// natif_test.go — DecodeNatif sur des tampons synthetiques et sur de vraies regions des
// blobs Forge verses en testdata (une region hkaiNavMesh d'un blob, posee derriere un
// en-tete `ucsh` synthetique, a la maniere d'une ressource de module).

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

const assetNatifTemoin = "01af558d-53ab-4f05-ba68-92d805fc6260"

// enteteNatif est un en-tete `ucsh` synthetique : signature puis une zone opaque.
func enteteNatif() []byte {
	return append([]byte(signatureNative), bytes.Repeat([]byte{0xAB}, 96)...)
}

// regionMaillageTemoin rend la region hkaiNavMesh du blob et les regions qui n'en portent pas.
func regionMaillageTemoin(t *testing.T, assetID string) (maillage []byte, autres [][]byte) {
	t.Helper()
	for _, r := range regionsTemoin(t, assetID) {
		if len(r) < 8 || string(r[4:8]) != sectionTAG0 {
			continue
		}
		_, vus, err := maillagesDeRegion(r)
		if err != nil {
			t.Fatalf("maillagesDeRegion: %v", err)
		}
		if len(vus) > 0 && maillage == nil {
			maillage = r
			continue
		}
		autres = append(autres, r)
	}
	if maillage == nil {
		t.Fatal("aucune region hkaiNavMesh dans le temoin")
	}
	return maillage, autres
}

func TestDecodeNatifSignatureAbsente(t *testing.T) {
	for nom, tampon := range map[string][]byte{
		"vide":            nil,
		"trop court":      []byte("uc"),
		"autre signature": append([]byte("xxxx"), make([]byte, 64)...),
	} {
		t.Run(nom, func(t *testing.T) {
			if _, err := DecodeNatif(tampon); !errors.Is(err, ErrSignatureNative) {
				t.Fatalf("erreur %v, ErrSignatureNative attendue", err)
			}
		})
	}
}

func TestDecodeNatifSansRegion(t *testing.T) {
	_, err := DecodeNatif(append(enteteNatif(), make([]byte, 256)...))
	if !errors.Is(err, ErrSansRegion) {
		t.Fatalf("erreur %v, ErrSansRegion attendue", err)
	}
}

func TestDecodeNatifSectionDebordante(t *testing.T) {
	// Un TAG0 dont la taille declaree depasse la ressource : erreur, pas region ignoree.
	tampon := enteteNatif()
	section := make([]byte, 16)
	binary.BigEndian.PutUint32(section, 0x40000000|1<<20) // drapeau + 1 Mo
	copy(section[4:], sectionTAG0)
	_, err := DecodeNatif(append(tampon, section...))
	if err == nil || errors.Is(err, ErrSansRegion) || errors.Is(err, ErrSansMaillage) {
		t.Fatalf("une section debordante doit etre une erreur propre, obtenu %v", err)
	}
}

func TestDecodeNatifSectionTropPetite(t *testing.T) {
	section := make([]byte, 16)
	binary.BigEndian.PutUint32(section, 4)
	copy(section[4:], sectionTAG0)
	if _, err := DecodeNatif(append(enteteNatif(), section...)); err == nil {
		t.Fatal("une taille de section inferieure a son en-tete doit etre refusee")
	}
}

func TestDecodeNatifRegionIllisible(t *testing.T) {
	// Bien delimite (taille exacte) mais sans DATA : lireFichierTag refuse, l'erreur remonte.
	section := make([]byte, 16)
	binary.BigEndian.PutUint32(section, uint32(len(section)))
	copy(section[4:], sectionTAG0)
	_, err := DecodeNatif(append(enteteNatif(), section...))
	if err == nil || errors.Is(err, ErrSansMaillage) {
		t.Fatalf("un fichier-tag illisible doit remonter son erreur, obtenu %v", err)
	}
}

func TestDecodeNatifAucunMaillage(t *testing.T) {
	_, autres := regionMaillageTemoin(t, assetNatifTemoin)
	if len(autres) == 0 {
		t.Skip("le temoin ne porte aucune region fichier-tag sans maillage")
	}
	tampon := enteteNatif()
	for _, r := range autres {
		tampon = append(tampon, r...)
	}
	if _, err := DecodeNatif(tampon); !errors.Is(err, ErrSansMaillage) {
		t.Fatalf("erreur %v, ErrSansMaillage attendue", err)
	}
}

func TestDecodeNatifMaillageUnique(t *testing.T) {
	maillage, autres := regionMaillageTemoin(t, assetNatifTemoin)
	tampon := enteteNatif()
	// Les fichiers-tag voisins sont inclus, comme dans une vraie ressource : seul le type
	// designe le maillage, pas la position.
	for _, r := range autres {
		tampon = append(tampon, r...)
	}
	tampon = append(tampon, maillage...)
	natif, err := DecodeNatif(tampon)
	if err != nil {
		t.Fatalf("DecodeNatif: %v", err)
	}
	ref := decodeTemoin(t, assetNatifTemoin)
	if len(natif.Sommets) != len(ref.Sommets) || len(natif.Faces) != len(ref.Faces) {
		t.Fatalf("natif %d sommets %d faces, blob %d sommets %d faces",
			len(natif.Sommets), len(natif.Faces), len(ref.Sommets), len(ref.Faces))
	}
	if natif.Min != ref.Min || natif.Max != ref.Max {
		t.Errorf("emprise differente: natif %v-%v, blob %v-%v", natif.Min, natif.Max, ref.Min, ref.Max)
	}
}

func TestDecodeNatifMaillagesMultiples(t *testing.T) {
	maillage, _ := regionMaillageTemoin(t, assetNatifTemoin)
	tampon := append(enteteNatif(), maillage...)
	tampon = append(tampon, maillage...)
	if _, err := DecodeNatif(tampon); !errors.Is(err, ErrMaillagesMultiples) {
		t.Fatalf("erreur %v, ErrMaillagesMultiples attendue", err)
	}
}

func TestDecodeNatifMasqueLesDrapeaux(t *testing.T) {
	// Les 2 bits de poids fort du u32 d'en-tete de section sont des drapeaux : un fichier-tag
	// valide dont l'en-tete porte 0x80000000|taille doit se decoder comme sans drapeau. Le bit 0x40000000
	// (feuille, lu par parcoursSections) ne convient pas sur la racine : il en ferait une feuille.
	maillage, _ := regionMaillageTemoin(t, assetNatifTemoin)
	region := append([]byte(nil), maillage...)
	taille := binary.BigEndian.Uint32(region)
	if taille&masqueTailleSection != uint32(len(region)) {
		t.Fatalf("la taille d'en-tete %d ne correspond pas a la region (%d octets)", taille&masqueTailleSection, len(region))
	}
	binary.BigEndian.PutUint32(region, 0x80000000|uint32(len(region)))
	natif, err := DecodeNatif(append(enteteNatif(), region...))
	if err != nil {
		t.Fatalf("DecodeNatif avec drapeau: %v", err)
	}
	if ref := decodeTemoin(t, assetNatifTemoin); len(natif.Faces) != len(ref.Faces) {
		t.Errorf("%d faces, %d attendues", len(natif.Faces), len(ref.Faces))
	}
}
