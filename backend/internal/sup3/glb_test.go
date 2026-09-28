package sup3

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func fixtureGLB(t *testing.T) []byte {
	t.Helper()
	bin := make([]byte, 36+24+128)
	for i, v := range []float32{0, 0, 0, 1, 0, 0, 0, 1, 0} {
		binary.LittleEndian.PutUint32(bin[i*4:], math.Float32bits(v))
	}
	for i := 0; i < 3; i++ {
		bin[36+i*8] = 0
		bin[36+i*8+1] = 1
		bin[36+i*8+4] = 128
		bin[36+i*8+5] = 127
	}
	for i := 0; i < 32; i++ {
		if (i%16)%5 == 0 {
			binary.LittleEndian.PutUint32(bin[60+i*4:], math.Float32bits(1))
		}
	}
	var root map[string]any
	err := json.Unmarshal([]byte(`{"asset":{"version":"2.0"},"buffers":[{"byteLength":124}],"bufferViews":[{"buffer":0,"byteOffset":0,"byteLength":36},{"buffer":0,"byteOffset":36,"byteLength":24,"byteStride":8},{"buffer":0,"byteOffset":60,"byteLength":64}],"accessors":[{"bufferView":0,"componentType":5126,"count":3,"type":"VEC3"},{"bufferView":1,"componentType":5121,"count":3,"type":"VEC4"},{"bufferView":1,"byteOffset":4,"componentType":5121,"normalized":true,"count":3,"type":"VEC4"},{"bufferView":2,"componentType":5126,"count":1,"type":"MAT4"}],"materials":[{"pbrMetallicRoughness":{"metallicFactor":0.2,"roughnessFactor":0.8}}],"meshes":[{"primitives":[{"attributes":{"POSITION":0,"JOINTS_0":1,"WEIGHTS_0":2},"material":0}]}],"nodes":[{"mesh":0,"skin":0},{"name":"root","children":[2]},{"name":"joint"}],"skins":[{"joints":[1,2],"inverseBindMatrices":3,"skeleton":1}],"scenes":[{"nodes":[0,1]}],"scene":0}`), &root)
	if err != nil {
		t.Fatal(err)
	}
	object(list(root["buffers"])[0])["byteLength"] = float64(len(bin))
	object(list(root["bufferViews"])[2])["byteLength"] = float64(128)
	object(list(root["accessors"])[3])["count"] = float64(2)
	b, e := marshalGLB(root, bin)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestExtractRealAccessorSemantics(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.glb")
	b := fixtureGLB(t)
	if e := os.WriteFile(src, b, 0600); e != nil {
		t.Fatal(e)
	}
	art, c, e := ExtractComponents(Artifact{ID: "a1", Path: src}, dir)
	if e != nil {
		t.Fatal(e)
	}
	for _, k := range []string{"geometry", "materials", "skeleton", "skin_weights"} {
		if c[k].Status != "available" {
			t.Fatalf("%s %+v", k, c[k])
		}
	}
	if c["animations"].Status != "absent" {
		t.Fatal("invented animation")
	}
	for _, a := range art {
		data, _ := os.ReadFile(a.Path)
		if a.Role == "skin_weights" {
			var m map[string]any
			_ = json.Unmarshal(data, &m)
			binding := object(list(m["bindings"])[0])
			set := object(list(binding["sets"])[0])
			w := list(list(set["weights"])[0])
			if math.Abs(w[0].(float64)-128.0/255) > 1e-8 || integer(binding["skin_index"]) != 0 {
				t.Fatal("normalized/interleaved weights incorrect")
			}
		}
		if a.Role == "geometry" {
			g, e := parseGLB(data)
			if e != nil {
				t.Fatal(e)
			}
			if g.Root["skins"] != nil || g.Root["materials"] != nil {
				t.Fatal("bindings not removed")
			}
		}
	}
	after, _ := os.ReadFile(src)
	if string(after) != string(b) {
		t.Fatal("source mutated")
	}
}
func TestSparseAndBounds(t *testing.T) {
	var root map[string]any
	_ = json.Unmarshal([]byte(`{"bufferViews":[{"buffer":0,"byteOffset":0,"byteLength":1},{"buffer":0,"byteOffset":4,"byteLength":4}],"accessors":[{"componentType":5126,"count":3,"type":"SCALAR","sparse":{"count":1,"indices":{"bufferView":0,"componentType":5121},"values":{"bufferView":1}}}]}`), &root)
	b := make([]byte, 8)
	b[0] = 1
	binary.LittleEndian.PutUint32(b[4:], math.Float32bits(0.75))
	d := glbDocument{Root: root, Bin: b}
	v, e := d.accessor(0)
	if e != nil || v[1][0] != 0.75 || v[0][0] != 0 {
		t.Fatalf("%v %v", v, e)
	}
	d.Bin = b[:4]
	if _, e = d.accessor(0); e == nil {
		t.Fatal("out of bounds accepted")
	}
	if _, e = parseGLB([]byte("glTF")); e == nil {
		t.Fatal("truncated header accepted")
	}
}
func TestPrivateDownloadAddressesRejected(t *testing.T) {
	for _, raw := range []string{"http://example.com/a", "https://127.0.0.1/a", "https://[::1]/a", "https://user:secret@example.com/a", "https://192.168.0.1/a", "https://169.254.169.254/a"} {
		if validRemoteURL(raw) == nil {
			t.Fatal(raw)
		}
	}
}

type fixtureTransport struct{ body []byte }

func (f fixtureTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, ContentLength: int64(len(f.body)), Body: io.NopCloser(bytes.NewReader(f.body)), Header: make(http.Header)}, nil
}
func TestAnimationContainerGetsComponentExtraction(t *testing.T) {
	e := NewEngine(nil, t.TempDir())
	e.DownloadClient = &http.Client{Transport: fixtureTransport{fixtureGLB(t)}}
	j := &Job{ID: "job_fixture", Components: map[string]Component{}, Artifacts: []Artifact{{ID: "a1", Role: "animation", Format: "glb", SourceURL: "https://assets.example.com/animated.glb"}}}
	if err := e.download(context.Background(), j); err != nil {
		t.Fatal(err)
	}
	if j.Components["skin_weights"].Status != "available" || j.Components["geometry"].Status != "available" {
		t.Fatal(j.Components)
	}
	for _, a := range j.Artifacts {
		if a.Size <= 0 || a.SHA256 == "" || a.URL == "" {
			t.Fatalf("undeliverable artifact %+v", a)
		}
	}
}
