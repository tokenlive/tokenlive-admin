package upgradehost

import (
	"context"
	"testing"
)

type fakeHost struct {
	cap Capability
	err error
}

func (f *fakeHost) Capability(context.Context) (Capability, error) { return f.cap, f.err }
func (f *fakeHost) Prepare(context.Context, PrepareRequest) (Preparation, error) {
	return Preparation{}, f.err
}
func (f *fakeHost) Submit(context.Context, SubmitRequest) (TaskView, error) {
	return TaskView{}, f.err
}
func (f *fakeHost) Task(context.Context, TaskRequest) (TaskView, error) {
	return TaskView{}, f.err
}

func TestRegisterCurrent(t *testing.T) {
	t.Cleanup(func() { Register(nil) })
	if Current() != nil {
		t.Fatal("expected no host before registration")
	}
	host := &fakeHost{}
	Register(host)
	if Current() != host {
		t.Fatal("expected registered host to be returned")
	}
	Register(nil)
	if Current() != nil {
		t.Fatal("expected cleared host")
	}
}

func TestCapabilityDefaultsRenderUnsupported(t *testing.T) {
	// The zero value must be a safe, honest "unsupported" answer so a
	// misbehaving host cannot accidentally enable the upgrade button.
	var cap Capability
	if cap.Supported || cap.Allowed {
		t.Fatal("zero capability must not claim support")
	}
	if len(cap.Reasons) != 0 {
		t.Fatalf("unexpected reasons: %v", cap.Reasons)
	}
	if cap.ActiveTask != nil {
		t.Fatal("unexpected active task")
	}
}
