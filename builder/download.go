package builder

import (
	"context"
	"fmt"
	"github.com/linode/linodego"
	"github.com/xyproto/randomstring"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// TODO: Eliminate code redundancy

func (b *Imagizer) DownloadImage(region, builderType string, volumeId int) error {
	timestamp := time.Now().Unix()

	log.Printf("Getting volume...\n")
	volume, err := b.client.GetVolume(context.Background(), volumeId)
	if err != nil {
		return fmt.Errorf("failed to retrieve volume: %v", err)
	}

	if volume.Status != linodego.VolumeActive {
		return fmt.Errorf("volume is not in a usable state: %v", volume.Status)
	}

	stackscriptContent, err := ioutil.ReadFile("stackscript/download_image.sh")
	if err != nil {
		return fmt.Errorf("failed to read stackscript file: %v", err)
	}

	log.Printf("Creating stackscript...\n")
	stackscript, err := b.client.CreateStackscript(context.Background(), linodego.StackscriptCreateOptions{
		Label:       fmt.Sprintf("imagizer-script-%v", timestamp),
		Description: "Uploads a volume-imagizer image",
		Images:      []string{"linode/alpine3.14"},
		Script:      string(stackscriptContent),
	})
	if err != nil {
		return fmt.Errorf("failed to create stackscript: %v", err)
	}

	defer func() {
		log.Printf("Deleting StackScript...\n")
		if err := b.client.DeleteStackscript(context.Background(), stackscript.ID); err != nil {
			log.Println(fmt.Errorf("failed to delete stackscript: %v", err))
		}
	}()

	booted := false

	instCreateOptions := linodego.InstanceCreateOptions{
		Region:          region,
		Type:            builderType,
		Label:           fmt.Sprintf("imagizer-builder-%v", timestamp),
		Image: 			 "linode/alpine3.14",
		RootPass: 		 randomstring.CookieFriendlyString(64),
		Booted: 		 &booted,
		StackScriptID:   stackscript.ID,
		StackScriptData: map[string]string{
			"volume_filepath": volume.FilesystemPath,
		},
	}

	log.Printf("Creating builder instance...\n")
	instance, err := b.client.CreateInstance(context.Background(), instCreateOptions)
	if err != nil {
		return fmt.Errorf("failed to create builder instance: %v", err)
	}

	defer func() {
		log.Printf("Deleting builder...\n")
		if err := b.client.DeleteInstance(context.Background(), instance.ID); err != nil {
			log.Println(fmt.Errorf("failed to clean up instance: %v", err))
		}
	}()


	log.Printf("Mounting volume on builder...\n")
	volume, err = b.client.AttachVolume(context.Background(), volume.ID, &linodego.VolumeAttachOptions{
		LinodeID:           instance.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to attach volume: %v", err)
	}


	log.Printf("Booting builder...\n")
	if err := b.client.BootInstance(context.Background(), instance.ID, 0); err != nil {
		return fmt.Errorf("failed to boot builder: %v", err)
	}

	// Poll for image download
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60 * 20)*time.Second)
	defer cancel()

	ticker := time.NewTicker(5000 * time.Millisecond)
	defer ticker.Stop()

	log.Println("Waiting for image host to be available...")
	for {
		select {
		case <-ticker.C:
			response, err := http.Get(fmt.Sprintf("http://%s:8081/image.img", instance.IPv4[0]))
			if err != nil {
				continue
			}

			log.Println("Writing file image.img...")
			out, err := os.Create("image.img")
			if err != nil {
				return err
			}

			_, err = io.Copy(out, response.Body)

			response.Body.Close()
			out.Close()

			return err

		case <-ctx.Done():
			return fmt.Errorf("error waiting for http server: context deadline exceeded")
		}
	}
}