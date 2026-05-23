package settings

import "testing"

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher("a-secret-of-at-least-sixteen-chars")
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	for _, in := range []string{
		"hello",
		"a-very-long-password-with-symbols-!@#$%^&*()",
		"বাংলা ইউনিকোডও কাজ করতে হবে",
	} {
		ct, err := c.Encrypt(in)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", in, err)
		}
		out, err := c.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if out != in {
			t.Fatalf("round-trip mismatch: %q vs %q", in, out)
		}
	}
}

func TestCipherShortKey(t *testing.T) {
	if _, err := NewCipher("short"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestCipherTamper(t *testing.T) {
	c, _ := NewCipher("a-secret-of-at-least-sixteen-chars")
	ct, _ := c.Encrypt("hello")
	// Flip the last char.
	bad := ct[:len(ct)-1] + string(rune(ct[len(ct)-1])^0x01)
	if _, err := c.Decrypt(bad); err == nil {
		t.Fatal("expected decrypt to fail on tampered ciphertext")
	}
}
