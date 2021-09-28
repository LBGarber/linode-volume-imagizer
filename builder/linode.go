package builder

import (
	"context"
	"fmt"
	"github.com/linode/linodego"
	"github.com/xyproto/randomstring"
	"io/ioutil"
	"log"
	"time"
)

// TODO: Eliminate code redundancy

func (b *Imagizer) BuildImage(region, builderType string, volumeId int) (*linodego.Image, error) {
	timestamp := time.Now().Unix()

	log.Printf("Getting volume...\n")
	volume, err := b.client.GetVolume(context.Background(), volumeId)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve volume: %v", err)
	}

	if volume.Status != linodego.VolumeActive {
		return nil, fmt.Errorf("volume is not in a usable state: %v", volume.Status)
	}

	stackscriptContent, err := ioutil.ReadFile("stackscript/upload_image.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to read stackscript file: %v", err)
	}

	log.Printf("Creating stackscript...\n")
	stackscript, err := b.client.CreateStackscript(context.Background(), linodego.StackscriptCreateOptions{
		Label:       fmt.Sprintf("imagizer-script-%v", timestamp),
		Description: "Uploads a volume-imagizer image",
		Images:      []string{"linode/alpine3.14"},
		Script:      string(stackscriptContent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create stackscript: %v", err)
	}

	defer func() {
		log.Printf("Deleting StackScript...\n")
		if err := b.client.DeleteInstance(context.Background(), stackscript.ID); err != nil {
			log.Println(fmt.Errorf("failed to delete stackscript: %v", err))
		}
	}()


	log.Printf("Creating image upload...\n")
	image, imageUpload, err := b.client.CreateImageUpload(context.Background(), linodego.ImageCreateUploadOptions{
		Region:      region,
		Label:       fmt.Sprintf("imagizer-image-%v", timestamp),
		Description: "funny",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create image upload: %v", err)
	}


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
			"image_uploadurl": imageUpload,
		},
	}

	log.Printf("Creating builder instance...\n")
	instance, err := b.client.CreateInstance(context.Background(), instCreateOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create builder instance: %v", err)
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
		return nil, fmt.Errorf("failed to attach volume: %v", err)
	}


	log.Printf("Booting builder...\n")
	if err := b.client.BootInstance(context.Background(), instance.ID, 0); err != nil {
		return nil, fmt.Errorf("failed to boot builder: %v", err)
	}


	log.Printf("Waiting for image available...\n")
	if _, err := b.client.WaitForImageStatus(
		context.Background(), image.ID, linodego.ImageStatusAvailable, 60 * 20); err != nil {
		return nil, fmt.Errorf("failed to wait for image available: %v", err)
	}


	finalImage, err := b.client.GetImage(context.Background(), image.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get finalized image: %v", err)
	}

	log.Printf("Image completed building: %v\n", image.ID)

	return finalImage, nil
}
