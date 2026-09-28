package sup3

import (
	"encoding/json"
	"fmt"
	"strings"
)

var tripoOptions = map[string]bool{"texture_quality": true, "geometry_quality": true, "smart_low_poly": true, "generate_parts": true, "texture_version": true, "model_seed": true, "texture_seed": true, "image_seed": true, "negative_prompt": true, "auto_size": true, "rig_type": true, "spec": true, "animation_in_place": true}
var meshyOptions = map[string]bool{"geometry_resolution": true, "height_meters": true, "moderation": true, "texture_prompt": true}

func validateOptionTypes(r Request) error {
	for k, v := range r.ProviderOptions {
		var target any
		switch k {
		case "smart_low_poly", "generate_parts", "auto_size", "animation_in_place", "moderation":
			target = new(bool)
		case "model_seed", "texture_seed", "image_seed":
			target = new(int64)
		case "height_meters":
			target = new(float64)
		default:
			target = new(string)
		}
		if string(v) == "null" || json.Unmarshal(v, target) != nil {
			return invalid("invalid type for provider_options." + k)
		}
	}
	return nil
}

func (p *RemoteProvider) price(r *Request) (Price, error) {
	price := Price{Provider: p.ID, Unit: "credits", Multiplier: 1, AsOf: "2026-09-29", Kind: "estimate"}
	if err := validateOptionTypes(*r); err != nil {
		return price, err
	}
	gen := strings.HasSuffix(r.Operation, "to_3d")
	if p.ID == "meshy" {
		if err := validateOptions(*r, meshyOptions); err != nil {
			return price, err
		}
		price.Source = "https://docs.meshy.ai/api/pricing"
		base := 20.0
		if gen {
			switch r.Model {
			case "meshy-7.1", "meshy-6":
			case "meshy-6-lite", "meshy-t2":
				base = 5
			default:
				return price, invalid("unsupported Meshy generation model")
			}
		}
		resolution := r.Parameters.TextureResolution
		if resolution == "" && (gen || r.Operation == "retexture") && r.Textured() {
			resolution = "2k"
			r.Parameters.TextureResolution = resolution
		}
		if gen && r.Model == "meshy-6-lite" && resolution != "2k" {
			return price, invalid("meshy-6-lite supports 2k textures only")
		}
		if r.Model == "meshy-t2" {
			if r.Operation == "multi_image_to_3d" {
				return price, invalid("meshy-t2 does not support multi-image")
			}
			if r.Parameters.Topology == "quad" {
				return price, invalid("meshy-t2 supports triangle topology only")
			}
			if r.Parameters.TargetFaces > 15000 {
				return price, invalid("meshy-t2 target_faces maximum is 15000")
			}
		}
		if r.Parameters.TargetFaces != 0 && (r.Parameters.TargetFaces < 100 || r.Parameters.TargetFaces > 300000) {
			return price, invalid("Meshy target_faces must be 100-300000")
		}
		texture := 10.0
		if resolution == "8k" {
			texture = 15
		}
		switch r.Operation {
		case "text_to_3d", "image_to_3d", "multi_image_to_3d":
			price.Credits = base
			if r.Textured() {
				price.Credits += texture
			}
		case "retexture":
			price.Credits = texture
		case "rig":
			price.Credits = 5
		case "animate":
			ids, err := animationIDs(r.Parameters.Animations)
			if err != nil {
				return price, err
			}
			price.Credits = 3 * float64(len(ids))
		}
		quality := option(*r, "geometry_resolution", "standard")
		if quality != "standard" {
			if !gen || r.Model != "meshy-7.1" || (quality != "2k" && quality != "4k") {
				return price, invalid("geometry_resolution 2k/4k requires meshy-7.1 generation")
			}
			price.Credits += 5
		}
		if h := option(*r, "height_meters", 1.7); h <= 0 || h > 1000 {
			return price, invalid("height_meters must be positive and <=1000")
		}
	} else {
		if err := validateOptions(*r, tripoOptions); err != nil {
			return price, err
		}
		price.Source = "https://developers.tripo3d.ai/en/pricing"
		if gen {
			switch r.Model {
			case "v3.1-20260211", "v3.0-20250812", "v2.5-20250123":
			default:
				return price, invalid("unsupported Tripo generation model")
			}
		}
		quality := option(*r, "texture_quality", "standard")
		if quality != "standard" && quality != "fast" && quality != "detailed" && quality != "extreme" {
			return price, invalid("invalid texture_quality")
		}
		if quality == "fast" && r.Operation != "retexture" && option(*r, "texture_version", "") != "v3.5-20260815" {
			return price, invalid("fast texture requires texture_version v3.5-20260815")
		}
		parts := option(*r, "generate_parts", false)
		low := option(*r, "smart_low_poly", false)
		if parts && (r.Textured() || r.PBR() || r.Parameters.Topology == "quad" || low) {
			return price, invalid("generate_parts requires texture=false, pbr=false, triangle topology, smart_low_poly=false")
		}
		if f := r.Parameters.MaxFaces; f != 0 {
			limit := 1500000
			if r.Model == "v3.0-20250812" {
				limit = 1000000
			}
			if r.Model == "v2.5-20250123" {
				limit = 500000
			}
			if option(*r, "geometry_quality", "standard") == "detailed" {
				limit = 2000000
			}
			if r.Parameters.Topology == "quad" {
				limit = 150000
			}
			min := 1
			if low {
				min = 500
				limit = 20000
				if r.Parameters.Topology == "quad" {
					limit = 10000
				}
			}
			if f < min || f > limit {
				return price, invalid(fmt.Sprintf("Tripo max_faces must be %d-%d for selected options", min, limit))
			}
		}
		texture := 10.0
		if quality == "detailed" {
			texture = 20
		}
		if quality == "extreme" {
			texture = 30
		}
		switch r.Operation {
		case "text_to_3d":
			price.Credits = 10
		case "image_to_3d", "multi_image_to_3d":
			price.Credits = 20
		case "retexture":
			price.Credits = texture
		case "rig":
			price.Credits = 25
		case "animate":
			price.Credits = 10 * float64(len(r.Parameters.Animations))
		}
		if gen {
			if r.Textured() {
				price.Credits += texture
			}
			if parts {
				price.Credits += 20
			}
			if low {
				price.Credits += 10
			}
			if r.Parameters.Topology == "quad" {
				price.Credits += 5
			}
			geom := option(*r, "geometry_quality", "standard")
			if geom == "detailed" {
				if r.Model == "v2.5-20250123" {
					return price, invalid("detailed geometry requires Tripo v3")
				}
				price.Credits += 20
			} else if geom != "standard" {
				return price, invalid("invalid geometry_quality")
			}
		}
		usd := price.Credits * 0.01
		price.USD = &usd
	}
	return price, nil
}

func (p *RemoteProvider) payload(r Request, stage, previous string) (string, map[string]any, error) {
	body := map[string]any{}
	input := r.Inputs.UpstreamID
	if input == "" {
		input = r.Inputs.ModelURL
	}
	if p.ID == "tripo" {
		endpoint := ""
		switch r.Operation {
		case "text_to_3d":
			endpoint = "/generation/text-to-model"
			body["prompt"] = r.Inputs.Prompt
		case "image_to_3d":
			endpoint = "/generation/image-to-model"
			body["input"] = r.Inputs.Images[0]
		case "multi_image_to_3d":
			endpoint = "/generation/multiview-to-model"
			body["inputs"] = r.Inputs.Images
		case "retexture":
			endpoint = "/models/texture"
			body["input"] = input
			body["model"] = r.Model
			body["pbr"] = r.PBR()
			if r.Inputs.Prompt != "" {
				body["texture_prompt"] = map[string]any{"text": r.Inputs.Prompt}
			}
		case "rig":
			endpoint = "/animations/rig"
			body["input"] = input
			body["model"] = r.Model
			body["out_format"] = "glb"
		case "animate":
			endpoint = "/animations/retarget"
			body["input"] = input
			body["animations"] = r.Parameters.Animations
			body["out_format"] = "glb"
			body["bake_animation"] = true
			body["export_with_geometry"] = true
		}
		if strings.HasSuffix(r.Operation, "to_3d") {
			body["model"] = r.Model
			body["texture"] = r.Textured()
			body["pbr"] = r.PBR()
			if r.Parameters.MaxFaces > 0 {
				body["face_limit"] = r.Parameters.MaxFaces
			}
			if r.Parameters.Topology == "quad" {
				body["quad"] = true
			}
		}
		for k, raw := range r.ProviderOptions {
			var v any
			_ = json.Unmarshal(raw, &v)
			if k == "animation_in_place" {
				k = "animate_in_place"
			}
			body[k] = v
		}
		return endpoint, body, nil
	}
	endpoint := ""
	switch r.Operation {
	case "text_to_3d":
		endpoint = "/openapi/v2/text-to-3d"
		if stage == "refine" {
			body["mode"] = "refine"
			body["preview_task_id"] = previous
			body["enable_pbr"] = r.PBR()
			body["texture_resolution"] = r.Parameters.TextureResolution
			body["target_formats"] = []string{"glb", "fbx"}
			if prompt := option(r, "texture_prompt", ""); prompt != "" {
				body["texture_prompt"] = prompt
			}
			return endpoint, body, nil
		}
		body["mode"] = "preview"
		body["prompt"] = r.Inputs.Prompt
	case "image_to_3d":
		endpoint = "/openapi/v1/image-to-3d"
		body["image_url"] = r.Inputs.Images[0]
	case "multi_image_to_3d":
		endpoint = "/openapi/v1/multi-image-to-3d"
		body["image_urls"] = r.Inputs.Images
	case "retexture":
		endpoint = "/openapi/v1/retexture"
		body["ai_model"] = r.Model
		body["target_formats"] = []string{"glb", "fbx"}
		if r.Inputs.UpstreamID != "" {
			body["input_task_id"] = input
		} else {
			body["model_url"] = input
		}
		body["enable_pbr"] = r.PBR()
		body["texture_resolution"] = r.Parameters.TextureResolution
		if r.Inputs.Prompt != "" {
			body["text_style_prompt"] = r.Inputs.Prompt
		}
	case "rig":
		endpoint = "/openapi/v1/rigging"
		if r.Inputs.UpstreamID != "" {
			body["input_task_id"] = input
		} else {
			body["model_url"] = input
		}
		body["height_meters"] = option(r, "height_meters", 1.7)
	case "animate":
		endpoint = "/openapi/v1/animations"
		body["rig_task_id"] = input
		ids, err := animationIDs(r.Parameters.Animations)
		if err != nil {
			return "", nil, err
		}
		body["action_ids"] = ids
	}
	if strings.HasSuffix(r.Operation, "to_3d") {
		body["ai_model"] = r.Model
		body["target_formats"] = []string{"glb", "fbx"}
		if r.Model == "meshy-t2" {
			body["model_type"] = "smart-topology"
		}
		if r.Operation != "text_to_3d" {
			body["should_texture"] = r.Textured()
			body["enable_pbr"] = r.PBR()
			body["texture_resolution"] = r.Parameters.TextureResolution
		}
		if r.Parameters.TargetFaces > 0 {
			body["target_polycount"] = r.Parameters.TargetFaces
			body["should_remesh"] = true
		}
		if r.Parameters.Topology != "" {
			body["topology"] = r.Parameters.Topology
			body["should_remesh"] = true
		}
		if r.Parameters.Pose != "" {
			body["pose_mode"] = r.Parameters.Pose
		}
		if q := option(r, "geometry_resolution", "standard"); q != "standard" {
			body["geometry_resolution"] = q
		}
		if raw, ok := r.ProviderOptions["moderation"]; ok {
			body["moderation"] = json.RawMessage(raw)
		}
	}
	return endpoint, body, nil
}
