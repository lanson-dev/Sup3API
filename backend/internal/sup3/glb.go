package sup3

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type glbDocument struct {
	Root map[string]any
	Bin  []byte
}

func parseGLB(b []byte) (glbDocument, error) {
	d := glbDocument{}
	if len(b) < 20 || string(b[:4]) != "glTF" || binary.LittleEndian.Uint32(b[4:8]) != 2 || int(binary.LittleEndian.Uint32(b[8:12])) != len(b) {
		return d, errors.New("invalid GLB 2.0 header")
	}
	for off := 12; off < len(b); {
		if off+8 > len(b) {
			return d, errors.New("truncated GLB chunk")
		}
		n := int(binary.LittleEndian.Uint32(b[off:]))
		kind := binary.LittleEndian.Uint32(b[off+4:])
		off += 8
		if n < 0 || n > len(b)-off {
			return d, errors.New("invalid GLB chunk length")
		}
		switch kind {
		case 0x4e4f534a:
			if d.Root != nil {
				return d, errors.New("duplicate JSON chunk")
			}
			if err := json.Unmarshal(bytes.TrimRight(b[off:off+n], " \x00"), &d.Root); err != nil {
				return d, err
			}
		case 0x004e4942:
			if d.Bin != nil {
				return d, errors.New("multiple BIN chunks")
			}
			d.Bin = b[off : off+n]
		}
		off += n
	}
	if d.Root == nil {
		return d, errors.New("missing GLB JSON")
	}
	return d, nil
}
func list(v any) []any  { a, _ := v.([]any); return a }
func integer(v any) int { n, _ := v.(float64); return int(n) }
func indexed(v any, i int) (map[string]any, error) {
	a := list(v)
	if i < 0 || i >= len(a) {
		return nil, errors.New("glTF index out of bounds")
	}
	m := object(a[i])
	if m == nil {
		return nil, errors.New("invalid glTF object")
	}
	return m, nil
}
func (d glbDocument) view(i int) ([]byte, int, error) {
	v, e := indexed(d.Root["bufferViews"], i)
	if e != nil {
		return nil, 0, e
	}
	if integer(v["buffer"]) != 0 {
		return nil, 0, errors.New("external glTF buffer unsupported")
	}
	if object(v["extensions"])["EXT_meshopt_compression"] != nil {
		return nil, 0, errors.New("meshopt accessor decoding not available")
	}
	off, n := integer(v["byteOffset"]), integer(v["byteLength"])
	if off < 0 || n < 0 || off > len(d.Bin) || n > len(d.Bin)-off {
		return nil, 0, errors.New("bufferView bounds invalid")
	}
	return d.Bin[off : off+n], integer(v["byteStride"]), nil
}
func scalar(b []byte, typ int, normalized bool) float64 {
	var n float64
	switch typ {
	case 5120:
		n = float64(int8(b[0]))
		if normalized {
			return math.Max(-1, n/127)
		}
	case 5121:
		n = float64(b[0])
		if normalized {
			return n / 255
		}
	case 5122:
		n = float64(int16(binary.LittleEndian.Uint16(b)))
		if normalized {
			return math.Max(-1, n/32767)
		}
	case 5123:
		n = float64(binary.LittleEndian.Uint16(b))
		if normalized {
			return n / 65535
		}
	case 5125:
		n = float64(binary.LittleEndian.Uint32(b))
	case 5126:
		n = float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
	}
	return n
}
func (d glbDocument) accessor(i int) ([][]float64, error) {
	a, e := indexed(d.Root["accessors"], i)
	if e != nil {
		return nil, e
	}
	count := integer(a["count"])
	typ := integer(a["componentType"])
	width := map[string]int{"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT4": 16}[str(a["type"])]
	size := map[int]int{5120: 1, 5121: 1, 5122: 2, 5123: 2, 5125: 4, 5126: 4}[typ]
	if count < 0 || width == 0 || size == 0 || count > 2000000/width {
		return nil, errors.New("unsupported or oversized accessor")
	}
	norm, _ := a["normalized"].(bool)
	values := make([][]float64, count)
	for j := range values {
		values[j] = make([]float64, width)
	}
	read := func(b []byte, offset, stride, n int, put func(int, []float64)) error {
		if stride == 0 {
			stride = width * size
		}
		if stride < width*size || offset < 0 || offset > len(b) || n > 0 && (n-1) > (len(b)-offset-width*size)/stride || n > 0 && len(b)-offset < width*size {
			return errors.New("accessor exceeds bufferView")
		}
		for j := 0; j < n; j++ {
			row := make([]float64, width)
			for k := 0; k < width; k++ {
				row[k] = scalar(b[offset+j*stride+k*size:], typ, norm)
				if math.IsNaN(row[k]) || math.IsInf(row[k], 0) {
					return errors.New("nonfinite accessor value")
				}
			}
			put(j, row)
		}
		return nil
	}
	if raw, ok := a["bufferView"]; ok {
		b, stride, e := d.view(integer(raw))
		if e != nil {
			return nil, e
		}
		if e = read(b, integer(a["byteOffset"]), stride, count, func(i int, v []float64) { values[i] = v }); e != nil {
			return nil, e
		}
	}
	if sparse := object(a["sparse"]); sparse != nil {
		n := integer(sparse["count"])
		if n < 0 || n > count {
			return nil, errors.New("invalid sparse count")
		}
		indices := object(sparse["indices"])
		it := integer(indices["componentType"])
		is := map[int]int{5121: 1, 5123: 2, 5125: 4}[it]
		if is == 0 {
			return nil, errors.New("invalid sparse index type")
		}
		ib, _, e := d.view(integer(indices["bufferView"]))
		if e != nil {
			return nil, e
		}
		ioff := integer(indices["byteOffset"])
		if ioff < 0 || ioff > len(ib) || n > (len(ib)-ioff)/is {
			return nil, errors.New("sparse indices out of bounds")
		}
		idx := make([]int, n)
		last := -1
		for j := range idx {
			idx[j] = int(scalar(ib[ioff+j*is:], it, false))
			if idx[j] <= last || idx[j] >= count {
				return nil, errors.New("invalid sparse index")
			}
			last = idx[j]
		}
		sv := object(sparse["values"])
		vb, _, e := d.view(integer(sv["bufferView"]))
		if e != nil {
			return nil, e
		}
		if e = read(vb, integer(sv["byteOffset"]), 0, n, func(j int, v []float64) { values[idx[j]] = v }); e != nil {
			return nil, e
		}
	}
	return values, nil
}
func marshalGLB(root map[string]any, bin []byte) ([]byte, error) {
	j, e := json.Marshal(root)
	if e != nil {
		return nil, e
	}
	for len(j)%4 != 0 {
		j = append(j, ' ')
	}
	bin = append([]byte(nil), bin...)
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	b := make([]byte, 12)
	copy(b, "glTF")
	binary.LittleEndian.PutUint32(b[4:], 2)
	binary.LittleEndian.PutUint32(b[8:], uint32(12+8+len(j)+8+len(bin)))
	for _, c := range []struct {
		k uint32
		b []byte
	}{{0x4e4f534a, j}, {0x004e4942, bin}} {
		h := make([]byte, 8)
		binary.LittleEndian.PutUint32(h, uint32(len(c.b)))
		binary.LittleEndian.PutUint32(h[4:], c.k)
		b = append(b, h...)
		b = append(b, c.b...)
	}
	return b, nil
}

// ExtractComponents preserves the original; derivatives carry source indices so
// geometry, materials and skin bindings can be joined without guessing names.
func ExtractComponents(source Artifact, dir string) ([]Artifact, map[string]Component, error) {
	out := []Artifact{}
	components := map[string]Component{}
	b, e := os.ReadFile(source.Path)
	if e != nil {
		return nil, nil, e
	}
	d, e := parseGLB(b)
	if e != nil {
		return nil, nil, e
	}
	emit := func(role, format string, data []byte) (string, error) {
		id := source.ID + "_" + strings.ReplaceAll(role, ".", "_")
		p := filepath.Join(dir, id+"."+format)
		if e := os.WriteFile(p, data, 0600); e != nil {
			return "", e
		}
		a := Artifact{ID: id, Role: role, Format: format, MediaType: "application/json", Path: p, DerivedFrom: source.ID}
		if format == "glb" {
			a.MediaType = "model/gltf-binary"
		}
		if format == "png" || format == "jpeg" || format == "webp" {
			a.MediaType = "image/" + format
		}
		out = append(out, a)
		return id, nil
	}
	emitJSON := func(role string, value any) (string, error) {
		data, e := json.Marshal(value)
		if e != nil {
			return "", e
		}
		return emit(role, "json", data)
	}
	available := func(component, id string) {
		components[component] = Component{Status: "available", ArtifactIDs: []string{id}}
	}
	for _, role := range []string{"geometry", "materials", "textures", "skeleton", "skin_weights", "animations"} {
		components[role] = Component{Status: "absent"}
	}
	// Material manifests retain glTF texture indices, texCoord and channel semantics.
	imageRefs := []map[string]any{}
	textureIDs := []string{}
	for i, im := range list(d.Root["images"]) {
		m := object(im)
		var data []byte
		var err error
		if v, ok := m["bufferView"]; ok {
			data, _, err = d.view(integer(v))
		} else if uri := str(m["uri"]); strings.HasPrefix(uri, "data:") {
			parts := strings.SplitN(uri, ",", 2)
			if len(parts) == 2 {
				data, err = base64.StdEncoding.DecodeString(parts[1])
			}
		}
		ref := map[string]any{"image_index": i, "mime_type": m["mimeType"]}
		if err == nil && len(data) > 0 {
			format := strings.TrimPrefix(str(m["mimeType"]), "image/")
			if format != "png" && format != "jpeg" && format != "webp" {
				format = "bin"
			}
			id, err := emit(fmt.Sprintf("texture.image_%d", i), format, data)
			if err != nil {
				return nil, nil, err
			}
			ref["artifact_id"] = id
			textureIDs = append(textureIDs, id)
		} else {
			ref["status"] = "unsupported"
			ref["reason"] = "image is external or cannot be decoded"
		}
		imageRefs = append(imageRefs, ref)
	}
	if len(textureIDs) > 0 {
		components["textures"] = Component{Status: "available", ArtifactIDs: textureIDs}
	} else if len(imageRefs) > 0 {
		components["textures"] = Component{Status: "unsupported", Reason: "no embedded decodable images"}
	}
	if len(list(d.Root["materials"])) > 0 {
		id, e := emitJSON("materials", map[string]any{"schema_version": SchemaVersion, "materials": d.Root["materials"], "textures": d.Root["textures"], "samplers": d.Root["samplers"], "images": imageRefs, "channels": map[string]string{"baseColorTexture": "RGBA sRGB", "metallicRoughnessTexture": "G=roughness B=metallic", "normalTexture": "RGB tangent-space", "occlusionTexture": "R=occlusion", "emissiveTexture": "RGB sRGB"}})
		if e != nil {
			return nil, nil, e
		}
		available("materials", id)
	}
	if len(list(d.Root["skins"])) > 0 {
		skins := []map[string]any{}
		var extractErr error
		for i, s := range list(d.Root["skins"]) {
			skin := object(s)
			entry := map[string]any{"skin_index": i, "joints": skin["joints"], "skeleton": skin["skeleton"], "name": skin["name"]}
			if raw, ok := skin["inverseBindMatrices"]; ok {
				m, e := d.accessor(integer(raw))
				if e != nil {
					extractErr = e
					break
				}
				entry["inverse_bind_matrices"] = m
			}
			skins = append(skins, entry)
		}
		if extractErr != nil {
			components["skeleton"] = Component{Status: "unsupported", Reason: extractErr.Error()}
		} else {
			id, e := emitJSON("skeleton", map[string]any{"schema_version": SchemaVersion, "nodes": d.Root["nodes"], "skins": skins, "matrix_layout": "column-major", "coordinate_system": "glTF right-handed, Y-up, meters"})
			if e != nil {
				return nil, nil, e
			}
			available("skeleton", id)
		}
		bindings := []map[string]any{}
		for ni, n := range list(d.Root["nodes"]) {
			node := object(n)
			skin, has := node["skin"]
			if !has {
				continue
			}
			mi, ok := node["mesh"]
			if !ok {
				continue
			}
			mesh, e := indexed(d.Root["meshes"], integer(mi))
			if e != nil {
				extractErr = e
				break
			}
			for pi, p := range list(mesh["primitives"]) {
				prim := object(p)
				if object(prim["extensions"])["KHR_draco_mesh_compression"] != nil {
					extractErr = errors.New("Draco skin accessor decoding not available")
					break
				}
				attrs := object(prim["attributes"])
				sets := []map[string]any{}
				for si := 0; ; si++ {
					jr, jok := attrs[fmt.Sprintf("JOINTS_%d", si)]
					wr, wok := attrs[fmt.Sprintf("WEIGHTS_%d", si)]
					if !jok && !wok {
						break
					}
					if !jok || !wok {
						extractErr = errors.New("unpaired joint/weight attributes")
						break
					}
					j, e := d.accessor(integer(jr))
					if e != nil {
						extractErr = e
						break
					}
					w, e := d.accessor(integer(wr))
					if e != nil {
						extractErr = e
						break
					}
					if len(j) != len(w) {
						extractErr = errors.New("joint/weight vertex counts differ")
						break
					}
					sets = append(sets, map[string]any{"set": si, "joints": j, "weights": w})
				}
				if len(sets) > 0 {
					bindings = append(bindings, map[string]any{"node_index": ni, "mesh_index": integer(mi), "primitive_index": pi, "skin_index": integer(skin), "sets": sets})
				}
			}
		}
		if extractErr != nil {
			components["skin_weights"] = Component{Status: "unsupported", Reason: extractErr.Error()}
		} else if len(bindings) > 0 {
			id, e := emitJSON("skin_weights", map[string]any{"schema_version": SchemaVersion, "joint_index_space": "index into the referenced skin.joints array", "vertex_order": "original primitive POSITION accessor order", "bindings": bindings})
			if e != nil {
				return nil, nil, e
			}
			available("skin_weights", id)
		}
	}
	if len(list(d.Root["animations"])) > 0 {
		clips := []map[string]any{}
		var err error
		for i, v := range list(d.Root["animations"]) {
			a := object(v)
			samplers := []map[string]any{}
			for _, sv := range list(a["samplers"]) {
				s := object(sv)
				in, e := d.accessor(integer(s["input"]))
				if e != nil {
					err = e
					break
				}
				out, e := d.accessor(integer(s["output"]))
				if e != nil {
					err = e
					break
				}
				samplers = append(samplers, map[string]any{"times": in, "values": out, "interpolation": s["interpolation"]})
			}
			clips = append(clips, map[string]any{"animation_index": i, "name": a["name"], "channels": a["channels"], "samplers": samplers})
		}
		if err != nil {
			components["animations"] = Component{Status: "unsupported", Reason: err.Error()}
		} else {
			id, e := emitJSON("animations", map[string]any{"schema_version": SchemaVersion, "clips": clips})
			if e != nil {
				return nil, nil, e
			}
			available("animations", id)
		}
	}
	// Strip bindings only; original binary bytes stay intact to avoid accessor rewrites.
	if len(list(d.Root["meshes"])) > 0 {
		delete(d.Root, "materials")
		delete(d.Root, "textures")
		delete(d.Root, "images")
		delete(d.Root, "samplers")
		delete(d.Root, "skins")
		delete(d.Root, "animations")
		for _, n := range list(d.Root["nodes"]) {
			delete(object(n), "skin")
		}
		for _, m := range list(d.Root["meshes"]) {
			for _, p := range list(object(m)["primitives"]) {
				prim := object(p)
				delete(prim, "material")
				attrs := object(prim["attributes"])
				for k := range attrs {
					if strings.HasPrefix(k, "JOINTS_") || strings.HasPrefix(k, "WEIGHTS_") {
						delete(attrs, k)
					}
				}
			}
		}
		data, e := marshalGLB(d.Root, d.Bin)
		if e != nil {
			return nil, nil, e
		}
		id, e := emit("geometry", "glb", data)
		if e != nil {
			return nil, nil, e
		}
		available("geometry", id)
	}
	return out, components, nil
}
