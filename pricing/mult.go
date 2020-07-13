package pricing

type Mult struct {
	Delta float64
}

func (mult *Mult) Inc() float64 {
	mult.Delta = mult.Delta + 0.01
	return mult.Delta
}

func (mult *Mult) Dec() float64 {
	mult.Delta = mult.Delta - 0.01
	return mult.Delta
}

func (mult *Mult) Calibrate(value float64) float64 {
	out := value + mult.Delta
	if out < 1.01 {
		for {
			out = value + mult.Inc()
			if out >= 1.01 {
				break
			}
		}
	}
	if out > 1.10 {
		for {
			out = value + mult.Dec()
			if out <= 1.10 {
				break
			}
		}
	}
	return out
}
