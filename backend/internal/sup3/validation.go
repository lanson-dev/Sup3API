package sup3

import "strings"

// Reject unsupported combinations instead of silently dropping normalized fields.
func validateSemantics(r Request) error {
	gen := strings.HasSuffix(r.Operation, "to_3d")
	tex := gen || r.Operation == "retexture"
	if len(r.Parameters.Formats) > 0 {
		if r.Provider != "meshy" || !tex {
			return invalid("formats is supported for Meshy generation and retexture only")
		}
		seen := map[string]bool{}
		for _, f := range r.Parameters.Formats {
			if (f != "glb" && f != "fbx" && f != "obj" && f != "stl" && f != "usdz" && f != "3mf") || seen[f] {
				return invalid("formats must contain distinct glb, fbx, obj, stl, usdz or 3mf values")
			}
			seen[f] = true
		}
	}
	if !gen && (r.Parameters.TargetFaces != 0 || r.Parameters.MaxFaces != 0 || r.Parameters.Topology != "" || r.Parameters.Pose != "") {
		return invalid("geometry parameters require a generation operation")
	}
	if !tex && (r.Parameters.Texture != nil || r.Parameters.PBR != nil || r.Parameters.TextureResolution != "") {
		return invalid("texture parameters require generation or retexture")
	}
	if gen && (r.Inputs.JobID != "" || r.Inputs.ModelURL != "") {
		return invalid("generation does not accept a source model")
	}
	if r.Operation == "text_to_3d" && len(r.Inputs.Images) > 0 {
		return invalid("text_to_3d does not accept image inputs")
	}
	if (r.Operation == "image_to_3d" || r.Operation == "multi_image_to_3d") && r.Inputs.Prompt != "" {
		return invalid("this image generation adapter does not accept a text prompt")
	}
	if !gen && len(r.Inputs.Images) > 0 {
		return invalid("this operation does not accept images")
	}
	if (r.Operation == "rig" || r.Operation == "animate") && r.Inputs.Prompt != "" {
		return invalid("this operation does not accept a prompt")
	}
	if r.Operation == "retexture" && !r.Textured() {
		return invalid("retexture requires texture=true")
	}
	if !r.Textured() && r.Parameters.TextureResolution != "" {
		return invalid("texture_resolution requires texture=true")
	}
	for _, image := range r.Inputs.Images {
		if image == "" && !(r.Provider == "tripo" && r.Operation == "multi_image_to_3d") {
			return invalid("image URL cannot be empty")
		}
	}
	for name := range r.ProviderOptions {
		allowed := false
		if r.Provider == "meshy" {
			switch name {
			case "geometry_resolution", "moderation":
				allowed = gen
			case "height_meters":
				allowed = r.Operation == "rig"
			case "texture_prompt":
				allowed = r.Operation == "text_to_3d" && r.Textured()
			}
		}
		if r.Provider == "tripo" {
			switch name {
			case "texture_quality":
				allowed = tex && r.Textured()
			case "geometry_quality", "smart_low_poly", "generate_parts", "model_seed", "auto_size":
				allowed = gen
			case "texture_version", "texture_seed":
				allowed = gen && r.Textured()
			case "image_seed", "negative_prompt":
				allowed = r.Operation == "text_to_3d"
			case "rig_type", "spec":
				allowed = r.Operation == "rig"
			case "animation_in_place":
				allowed = r.Operation == "animate"
			}
		}
		if !allowed {
			return invalid("provider_options." + name + " is unsupported for this operation")
		}
	}
	if r.Provider == "meshy" {
		if (r.Operation == "rig" || r.Operation == "animate") && r.Model != "" {
			return invalid("Meshy rig/animate do not accept model versions")
		}
		if r.Operation == "retexture" && r.Model != "meshy-7" && r.Model != "meshy-6" && r.Model != "meshy-6-lite" {
			return invalid("unsupported Meshy retexture model")
		}
	} else {
		if r.Operation == "animate" && r.Model != "" {
			return invalid("Tripo animate inherits the rig model")
		}
		if r.Operation == "rig" {
			if r.Model != "v1.0-20240301" && r.Model != "v2.5-20260210" {
				return invalid("unsupported Tripo rig model")
			}
			rig := option(r, "rig_type", "biped")
			if rig != "biped" && rig != "quadruped" && rig != "hexapod" && rig != "octopod" && rig != "avian" && rig != "serpentine" && rig != "aquatic" {
				return invalid("unsupported rig_type")
			}
			if r.Model == "v1.0-20240301" && rig != "biped" {
				return invalid("Tripo v1 rig supports biped only")
			}
			spec := option(r, "spec", "tripo")
			if spec != "tripo" && spec != "mixamo" {
				return invalid("unsupported rig spec")
			}
		}
		if r.Operation == "retexture" && r.Model != "v3.5-20260815" {
			return invalid("unsupported Tripo texture model")
		}
		if v := option(r, "texture_version", ""); v != "" && v != "v3.5-20260815" {
			return invalid("unsupported texture_version")
		}
		seen := map[string]bool{}
		for _, a := range r.Parameters.Animations {
			if strings.TrimSpace(a) == "" || seen[a] {
				return invalid("animations must be nonempty and distinct")
			}
			seen[a] = true
		}
	}
	return nil
}
