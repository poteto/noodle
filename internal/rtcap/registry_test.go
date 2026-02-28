package rtcap

import "testing"

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()

	caps, ok := reg.Get("process")
	if !ok {
		t.Fatal("process runtime not found in registry")
	}
	if caps.Name != "process" {
		t.Errorf("process name = %q, want %q", caps.Name, "process")
	}
	if !caps.Has(CapSteerable) {
		t.Error("process should be steerable")
	}

	caps, ok = reg.Get("sprites")
	if !ok {
		t.Fatal("sprites runtime not found in registry")
	}
	if !caps.Has(CapSteerable) {
		t.Error("sprites should be steerable")
	}

	caps, ok = reg.Get("cursor")
	if !ok {
		t.Fatal("cursor runtime not found in registry")
	}
	if !caps.Has(CapPolling) {
		t.Error("cursor should need polling")
	}
	if !caps.Has(CapRemoteSync) {
		t.Error("cursor should need remote sync")
	}
	if caps.Has(CapSteerable) {
		t.Error("cursor should not be steerable")
	}
}

func TestRegistryGetNormalizesName(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("  PROCESS  ")
	if !ok {
		t.Error("Get should normalize name case and whitespace")
	}
}

func TestRegistryGetUnknown(t *testing.T) {
	reg := NewRegistry()
	_, ok := reg.Get("unknown-runtime")
	if ok {
		t.Error("Get should return false for unknown runtime")
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()
	custom := RuntimeCapabilities{
		Name: "custom",
		Caps: map[Capability]bool{
			CapHeartbeat: true,
			CapPolling:   true,
		},
	}
	reg.Register("Custom", custom)

	got, ok := reg.Get("custom")
	if !ok {
		t.Fatal("custom runtime not found after Register")
	}
	if !got.Has(CapHeartbeat) {
		t.Error("custom should have heartbeat")
	}
	if !got.Has(CapPolling) {
		t.Error("custom should have polling")
	}
	if got.Has(CapSteerable) {
		t.Error("custom should not have steerable")
	}
}

func TestRegistryRegisterOverwrite(t *testing.T) {
	reg := NewRegistry()

	// Overwrite process with different caps.
	reg.Register("process", RuntimeCapabilities{
		Caps: map[Capability]bool{
			CapPolling: true,
		},
	})

	got, ok := reg.Get("process")
	if !ok {
		t.Fatal("process not found after overwrite")
	}
	if !got.Has(CapPolling) {
		t.Error("overwritten process should have polling")
	}
	if got.Has(CapSteerable) {
		t.Error("overwritten process should not have steerable")
	}
}

func TestRegistryDefaultCapabilities(t *testing.T) {
	reg := NewRegistry()
	def := reg.DefaultCapabilities()
	if def.Name != "process" {
		t.Errorf("DefaultCapabilities().Name = %q, want %q", def.Name, "process")
	}
	if !def.Has(CapSteerable) {
		t.Error("default should be steerable (process profile)")
	}
}
