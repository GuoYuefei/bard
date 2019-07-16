package debug

import "fmt"

type Plugin struct {
	VVersion string
	Id string
	PPriority uint16
	End byte
}

func (p Plugin)EndCam() byte {
	return p.End
}

func (p Plugin)Camouflage(bs []byte, send bool) ([]byte, int) {
	fmt.Printf("%s: Camouflage\t %x\n", p.Id, p.PPriority)
	return bs, len(bs)
}

func (p Plugin)AntiSniffing(bs []byte, send bool) ([]byte, int) {
	fmt.Printf("%s: AntiSniffing\t %x\n", p.Id, p.PPriority)
	return bs, len(bs)
}
func (p Plugin)Ornament(bs []byte, send bool) ([]byte, int) {
	fmt.Printf("%s: Ornament\t %x\n", p.Id, p.PPriority)
	return bs, len(bs)
}
func (p Plugin)Priority() uint16 {
	return p.PPriority
}
func (p Plugin)GetID() string {
	return p.Id
}
func (p Plugin)Version() string {
	return p.VVersion
}

