package sup3

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

// Native envelopes reuse the same validation, ownership, quotes and idempotency
// as unified requests. They are a field adapter, not an unrestricted proxy.
func normalizeInput(r Request) (Request, error) {
	if r.InputFormat == "" || r.InputFormat == "agraphs" {
		if r.Payload != nil {
			return r, invalid("payload requires input_format tripo or meshy")
		}
		r.InputFormat = ""
		return r, nil
	}
	if (r.InputFormat != "tripo" && r.InputFormat != "meshy") || r.Provider != r.InputFormat {
		return r, invalid("input_format must match provider: tripo or meshy")
	}
	if r.Payload == nil || r.Model != "" || r.Inputs.Prompt != "" || len(r.Inputs.Images) > 0 || r.Inputs.ModelURL != "" || r.Inputs.JobID != "" || len(r.ProviderOptions) > 0 || r.Parameters.Texture != nil || r.Parameters.PBR != nil || r.Parameters.TextureResolution != "" || r.Parameters.TargetFaces != 0 || r.Parameters.MaxFaces != 0 || r.Parameters.Topology != "" || r.Parameters.Pose != "" || len(r.Parameters.Animations) > 0 || len(r.Parameters.Formats) > 0 {
		return r, invalid("native payload cannot be mixed with unified model, inputs, parameters or provider_options")
	}
	out := Request{Provider: r.Provider, Operation: r.Operation, ProviderOptions: map[string]json.RawMessage{}}
	fields := map[string]any{}
	gen := r.Operation == "text_to_3d" || r.Operation == "image_to_3d" || r.Operation == "multi_image_to_3d"
	if r.Provider == "tripo" {
		if gen || r.Operation == "retexture" || r.Operation == "rig" {
			fields["model"] = &out.Model
		}
		if gen {
			fields["texture"], fields["pbr"], fields["face_limit"] = &out.Parameters.Texture, &out.Parameters.PBR, &out.Parameters.MaxFaces
		}
		switch r.Operation {
		case "text_to_3d":
			fields["prompt"] = &out.Inputs.Prompt
		case "image_to_3d":
			var value string
			if err := nativeField(r.Payload, "input", &value); err != nil {
				return out, err
			}
			out.Inputs.Images = []string{value}
			fields["input"] = &value
		case "multi_image_to_3d":
			fields["inputs"] = &out.Inputs.Images
		case "retexture", "rig", "animate":
			var value string
			if err := nativeField(r.Payload, "input", &value); err != nil {
				return out, err
			}
			if strings.HasPrefix(value, "job_") {
				out.Inputs.JobID = value
			} else {
				out.Inputs.ModelURL = value
			}
			fields["input"] = &value
			if r.Operation == "retexture" {
				fields["pbr"] = &out.Parameters.PBR
				if raw, ok := r.Payload["texture_prompt"]; ok {
					var prompt struct {
						Text string `json:"text"`
					}
					d := json.NewDecoder(bytes.NewReader(raw))
					d.DisallowUnknownFields()
					if string(raw) == "null" || d.Decode(&prompt) != nil {
						return out, invalid("texture_prompt supports {text: string}")
					}
					out.Inputs.Prompt = prompt.Text
					fields["texture_prompt"] = &prompt
				}
			}
			if r.Operation == "animate" {
				fields["animations"] = &out.Parameters.Animations
			}
		}
		if _, ok := r.Payload["quad"]; ok && gen {
			var quad bool
			if err := nativeField(r.Payload, "quad", &quad); err != nil {
				return out, err
			}
			if quad {
				out.Parameters.Topology = "quad"
			} else {
				out.Parameters.Topology = "triangle"
			}
			fields["quad"] = &quad
		}
	} else {
		if gen || r.Operation == "retexture" {
			fields["ai_model"] = &out.Model
			fields["target_formats"] = &out.Parameters.Formats
		}
		if gen {
			fields["target_polycount"], fields["topology"], fields["pose_mode"] = &out.Parameters.TargetFaces, &out.Parameters.Topology, &out.Parameters.Pose
		}
		if gen || r.Operation == "retexture" {
			pbr := false // Meshy's native default differs from AGraphs' unified default.
			out.Parameters.PBR = &pbr
			fields["enable_pbr"], fields["texture_resolution"] = &out.Parameters.PBR, &out.Parameters.TextureResolution
		}
		switch r.Operation {
		case "text_to_3d":
			fields["prompt"] = &out.Inputs.Prompt
			var mode string
			if err := nativeField(r.Payload, "mode", &mode); err != nil {
				return out, err
			}
			if mode != "preview" {
				return out, invalid("native text_to_3d requires mode=preview; use unified requests for automatic preview + refine")
			}
			texture := false
			out.Parameters.Texture = &texture
			fields["mode"] = &mode
		case "image_to_3d":
			var value string
			if err := nativeField(r.Payload, "image_url", &value); err != nil {
				return out, err
			}
			out.Inputs.Images = []string{value}
			fields["image_url"], fields["should_texture"] = &value, &out.Parameters.Texture
		case "multi_image_to_3d":
			fields["image_urls"], fields["should_texture"] = &out.Inputs.Images, &out.Parameters.Texture
		case "retexture", "rig":
			fields["model_url"] = &out.Inputs.ModelURL
			// References are gateway IDs, never caller-supplied upstream account IDs.
			fields["input_task_id"] = &out.Inputs.JobID
			if r.Operation == "retexture" {
				fields["text_style_prompt"] = &out.Inputs.Prompt
			}
		case "animate":
			fields["rig_task_id"] = &out.Inputs.JobID
			var ids []int
			if err := nativeField(r.Payload, "action_ids", &ids); err != nil {
				return out, err
			}
			for _, id := range ids {
				b, _ := json.Marshal(id)
				out.Parameters.Animations = append(out.Parameters.Animations, string(b))
			}
			fields["action_ids"] = &ids
		}
	}
	options := map[string]string{}
	if r.Provider == "tripo" {
		for _, k := range []string{"texture_quality", "geometry_quality", "smart_low_poly", "generate_parts", "model_seed", "auto_size", "texture_version", "texture_seed", "image_seed", "negative_prompt", "rig_type", "spec"} {
			options[k] = k
		}
		options["animate_in_place"] = "animation_in_place"
	} else {
		for _, k := range []string{"geometry_resolution", "moderation", "height_meters", "texture_prompt"} {
			options[k] = k
		}
	}
	for key, raw := range r.Payload {
		if dest, ok := fields[key]; ok {
			if err := nativeField(r.Payload, key, dest); err != nil {
				return out, err
			}
		} else if mapped, ok := options[key]; ok {
			out.ProviderOptions[mapped] = raw
		} else {
			return out, invalid("unsupported native payload field: " + key)
		}
	}
	if out.Inputs.JobID != "" && !strings.HasPrefix(out.Inputs.JobID, "job_") {
		return out, invalid("task references must be AGraphs job IDs owned by this API key")
	}
	return out, nil
}

func nativeField(payload map[string]json.RawMessage, key string, dest any) error {
	if raw, ok := payload[key]; ok {
		if string(raw) == "null" || json.Unmarshal(raw, dest) != nil {
			return invalid("invalid native payload field: " + key)
		}
	}
	return nil
}

func validateImageInput(provider, value string) error {
	if !strings.HasPrefix(value, "data:") {
		return validRemoteURL(value)
	}
	if provider != "meshy" {
		return invalid("Tripo image inputs require a public HTTPS URL")
	}
	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 || (parts[0] != "data:image/png;base64" && parts[0] != "data:image/jpeg;base64") {
		return invalid("Meshy data URI must be base64 PNG or JPEG")
	}
	b, err := base64.StdEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(b) == 0 || len(b) > 10<<20 {
		return invalid("invalid base64 image or image exceeds 10 MiB")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 16384 || cfg.Height > 16384 || parts[0] != "data:image/"+format+";base64" {
		return invalid("invalid image bytes, MIME type or dimensions (maximum 16384 per side)")
	}
	return nil
}
