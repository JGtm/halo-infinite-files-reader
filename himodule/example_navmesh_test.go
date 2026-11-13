package himodule_test

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/JGtm/halo-infinite-files-reader/himodule"
	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// Read the navigation mesh of a native map: find the resource that carries it in the map's
// module (the first one is the main mesh), extract it and decode it.
func ExampleModule_RessourcesNavmesh() {
	path := filepath.Join("deploy", "any", "levels", "multi", "ctf_bazaar", "ctf_bazaar-rtx-new.module")

	m, err := himodule.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	ressources, err := m.RessourcesNavmesh(context.Background())
	if err != nil {
		log.Printf("some candidate resources were unreadable: %v", err) // the readable ones are still returned
	}
	if len(ressources) == 0 {
		log.Fatal("this map has no navigation mesh")
	}
	brut, err := m.Extract(ressources[0])
	if err != nil {
		log.Fatal(err)
	}
	maillage, err := hinavmesh.DecodeNatif(brut)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d vertices, %d faces, walkable area %.0f m2\n",
		len(maillage.Sommets), len(maillage.Faces), maillage.AireAuSol())
}
