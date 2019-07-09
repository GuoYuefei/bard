package debug

import "fmt"

type Plugin struct {
	VVersion string
	Id string
	PPriority uint16
}

func (p Plugin)Camouflage([]byte) ([]byte, int) {
	fmt.Printf("%s: Camouflage\t %x\n", p.Id, p.PPriority)
	return []byte{}, 0
}

func (p Plugin)AntiSniffing([]byte) ([]byte, int) {
	fmt.Printf("%s: AntiSniffing\t %x\n", p.Id, p.PPriority)
	return []byte{}, 0
}
func (p Plugin)Ornament([]byte) ([]byte, int) {
	fmt.Printf("%s: Ornament\t %x\n", p.Id, p.PPriority)
	return []byte{}, 0
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

