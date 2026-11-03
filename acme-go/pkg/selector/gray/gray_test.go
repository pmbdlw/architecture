package gray

import (
	"reflect"
	"testing"
)

type node struct {
	id  string
	tag string
}

func meta(n node) map[string]string { return map[string]string{MetadataKey: n.tag} }

func ids(nodes []node) []string {
	var out []string
	for _, n := range nodes {
		out = append(out, n.id)
	}
	return out
}

func TestSelect(t *testing.T) {
	stable1, stable2, v2 := node{"s1", ""}, node{"s2", ""}, node{"g1", "v2"}
	cases := []struct {
		name  string
		nodes []node
		tag   string
		want  []string
	}{
		{"普通流量只走稳定实例", []node{stable1, stable2, v2}, "", []string{"s1", "s2"}},
		{"灰度流量命中同标签实例", []node{stable1, v2}, "v2", []string{"g1"}},
		{"无对应灰度实例时回落稳定实例", []node{stable1, v2}, "v3", []string{"s1"}},
		{"全部为灰度实例时不过滤", []node{v2}, "", []string{"g1"}},
	}
	for _, c := range cases {
		if got := ids(Select(c.nodes, c.tag, meta)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
