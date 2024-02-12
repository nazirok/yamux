package yamux

import "testing"

func Test_Config_Init(t *testing.T) {
	c := DefaultConfig()
	if c.MaxReceiveBuffer != minReceiveBuffer {
		t.Fatalf("wrong max recieve buffer: %d", c.MaxReceiveBuffer)
	}

	err := VerifyConfig(c)
	if err != nil {
		t.Fatalf("verify config err: %v", err)
	}

	c.MaxReceiveBuffer = 0
	err = VerifyConfig(c)
	if err != nil {
		t.Fatalf("verify config err: %v", err)
	}
	if c.MaxReceiveBuffer != minReceiveBuffer {
		t.Fatalf("bad receive buffer: %d", c.MaxReceiveBuffer)
	}

	c.MaxReceiveBuffer = 13123
	err = VerifyConfig(c)
	if err == nil {
		t.Fatal("verify config fail, must err: MaxReceiveBuffer in (0..1MB)")
	}

	c.MaxReceiveBuffer = 2 * 1024 * 1204
	err = VerifyConfig(c)
	if err != nil {
		t.Fatalf("verify config err: %v", err)
	}

	if c.MaxReceiveBuffer != 2*1024*1204 {
		t.Fatalf("bad receive buffer: %d", c.MaxReceiveBuffer)
	}
}
