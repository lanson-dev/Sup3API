package sup3

import (
	"encoding/json"
	"slices"
	"strings"
)

type OutputRequirements struct {
	Formats            []string `json:"formats,omitempty"`
	RequiredComponents []string `json:"required_components,omitempty"`
}

type OutputValidation struct {
	Status  string   `json:"status"`
	Missing []string `json:"missing"`
}

// Keep old parameters.formats requests working while giving new clients a
// provider-independent output contract and namespaced provider extensions.
func normalizeContract(r *Request) error {
	for provider, fields := range r.Extensions {
		if provider != r.Provider {
			return invalid("extensions must use the selected provider namespace")
		}
		if r.ProviderOptions == nil {
			r.ProviderOptions = map[string]json.RawMessage{}
		}
		for name, value := range fields {
			if _, exists := r.ProviderOptions[name]; exists {
				return invalid("extension conflicts with provider_options." + name)
			}
			r.ProviderOptions[name] = value
		}
	}
	r.Extensions = nil
	if r.Output == nil {
		return nil
	}
	if len(r.Output.Formats) > 0 {
		if len(r.Parameters.Formats) > 0 && !sameStrings(r.Output.Formats, r.Parameters.Formats) {
			return invalid("output.formats conflicts with parameters.formats")
		}
		r.Parameters.Formats = append([]string{}, r.Output.Formats...)
	}
	seen := map[string]bool{}
	for _, component := range r.Output.RequiredComponents {
		if !slices.Contains([]string{"geometry", "materials", "textures", "skeleton", "skin_weights", "animations"}, component) || seen[component] {
			return invalid("required_components must contain distinct geometry, materials, textures, skeleton, skin_weights or animations")
		}
		seen[component] = true
		if (component == "materials" || component == "textures") && !r.Textured() {
			return invalid("required materials/textures conflict with texture=false")
		}
		if (component == "skeleton" || component == "skin_weights") && r.Operation != "rig" && r.Operation != "animate" {
			return invalid("required skeleton/skin_weights need rig or animate")
		}
		if component == "animations" && r.Operation != "animate" {
			return invalid("required animations need animate")
		}
	}
	if len(seen) > 0 {
		if r.Provider == "tripo" && r.Parameters.Topology == "quad" {
			return invalid("component verification needs GLB; Tripo quad outputs FBX")
		}
		if len(r.Parameters.Formats) > 0 && !slices.Contains(r.Parameters.Formats, "glb") {
			return invalid("required component verification currently needs glb in output.formats")
		}
	}
	return nil
}

func sameStrings(a, b []string) bool {
	a = append([]string{}, a...)
	b = append([]string{}, b...)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func directFormats(provider, operation, topology string) []string {
	if provider == "tripo" {
		if strings.HasSuffix(operation, "to_3d") && topology == "quad" {
			return []string{"fbx"}
		}
		return []string{"glb"}
	}
	if operation == "rig" || operation == "animate" {
		return nil
	}
	return []string{"glb", "fbx", "obj", "stl", "usdz", "3mf"}
}

func validateOutput(j *Job) bool {
	if j.Request.Output == nil && len(j.Request.Parameters.Formats) == 0 {
		return true
	}
	result := &OutputValidation{Status: "passed", Missing: []string{}}
	for _, format := range j.Request.Parameters.Formats {
		found := slices.ContainsFunc(j.Artifacts, func(a Artifact) bool {
			return a.DerivedFrom == "" && (a.Role == "model" || a.Role == "animation") && a.Format == format
		})
		if !found {
			result.Missing = append(result.Missing, "format:"+format)
		}
	}
	if j.Request.Output != nil {
		for _, component := range j.Request.Output.RequiredComponents {
			if j.Components[component].Status != "available" {
				result.Missing = append(result.Missing, "component:"+component)
			}
		}
	}
	if len(result.Missing) > 0 {
		result.Status = "failed"
	}
	j.OutputValidation = result
	return result.Status == "passed"
}

func operationDetails(provider string, models map[string][]string) map[string]any {
	out := map[string]any{}
	for operation, versions := range models {
		detail := map[string]any{"models": versions, "selectable_output_formats": directFormats(provider, operation, "triangle"), "required_components_verification": "glb_only", "automatic_conversion": false}
		if provider == "tripo" && strings.HasSuffix(operation, "to_3d") {
			detail["format_conditions"] = map[string]any{"triangle": []string{"glb"}, "quad": []string{"fbx"}}
			detail["face_count_semantics"] = "maximum"
		} else if provider == "meshy" && strings.HasSuffix(operation, "to_3d") {
			detail["face_count_semantics"] = "approximate_target"
		}
		if operation == "text_to_3d" && provider == "meshy" {
			detail["stages"] = []string{"preview", "refine_if_texture_enabled"}
		}
		options := map[string]any{}
		allowed := meshyOptions
		if provider == "tripo" {
			allowed = tripoOptions
		}
		for name := range allowed {
			if optionApplies(Request{Provider: provider, Operation: operation}, name) {
				options[name] = extensionSchema(name)
			}
		}
		detail["extension_fields"] = options
		out[operation] = detail
	}
	return out
}

func extensionSchema(name string) map[string]any {
	s := map[string]any{"type": "string"}
	switch name {
	case "smart_low_poly", "generate_parts", "auto_size", "animation_in_place", "moderation":
		s["type"] = "boolean"
	case "model_seed", "texture_seed", "image_seed":
		s["type"] = "integer"
	case "height_meters":
		s["type"] = "number"
		s["exclusiveMinimum"] = 0
		s["maximum"] = 1000
	case "geometry_resolution":
		s["enum"] = []string{"standard", "2k", "4k"}
		s["description"] = "2k/4k require meshy-7.1"
	case "geometry_quality":
		s["enum"] = []string{"standard", "detailed"}
	case "texture_quality":
		s["enum"] = []string{"standard", "fast", "detailed", "extreme"}
		s["description"] = "Requires texture; fast generation requires texture_version v3.5-20260815"
	case "texture_version":
		s["enum"] = []string{"v3.5-20260815"}
	case "rig_type":
		s["enum"] = []string{"biped", "quadruped", "hexapod", "octopod", "avian", "serpentine", "aquatic"}
		s["description"] = "v1.0-20240301 supports biped only"
	case "spec":
		s["enum"] = []string{"tripo", "mixamo"}
	}
	if name == "generate_parts" {
		s["description"] = "Requires texture=false, pbr=false, triangle topology and smart_low_poly=false"
	}
	return s
}
