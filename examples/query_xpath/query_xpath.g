%axiom Query

%preamble ParserPreallocMem

%%

Query : Path {
	$$.Value = $1.Value
};

Path : Path CHILD Step {
	$$.Value = appendStep($1.Value.(*peSemValue), $3.Value.(udpeTest), child)
} | Path PARENT Step {
	$$.Value = appendStep($1.Value.(*peSemValue), $3.Value.(udpeTest), parent)
} | Path ANCESTOR Step {
	$$.Value = appendStep($1.Value.(*peSemValue), $3.Value.(udpeTest), ancestorOrSelf)
} | Path DESCENDANT Step {
	$$.Value = appendStep($1.Value.(*peSemValue), $3.Value.(udpeTest), descendantOrSelf)
} | CHILD Step {
	$$.Value = appendStep(nil, $2.Value.(udpeTest), child)
} | PARENT Step {
	$$.Value = appendStep(nil, $2.Value.(udpeTest), parent)
} | ANCESTOR Step {
	$$.Value = appendStep(nil, $2.Value.(udpeTest), ancestorOrSelf)
} | DESCENDANT Step {
	$$.Value = appendStep(nil, $2.Value.(udpeTest), descendantOrSelf)
};

Step : Test {}
  | Test LBR Path RBR {
	switch $1.Value.(type) {
	case *elementTest:
		$1.Value.(*elementTest).pred = buildPredicate(toExpr($3.Value), $1.Value.(*elementTest))
	case *textTest:
		$1.Value.(*elementTest).pred = nil
	default:
		panic("unknown test type")
	}
	$$.Value = $1.Value
} | Test LBR OrExpr RBR {
	// TODO(vvihorev): support NUDPEs inside predicates
	switch $1.Value.(type) {
	case *elementTest:
		$1.Value.(*elementTest).pred = buildPredicate(toExpr($3.Value), $1.Value.(*elementTest))
	case *textTest:
		$1.Value.(*elementTest).pred = nil
	default:
		panic("unknown test type")
	}
	$$.Value = $1.Value
};

Test : IDENT {
	$$.Value = newElementTest($1.Value.(string), nil, nil)
} | AT IDENT EQ STRING {
	$$.Value = newElementTest("*", &Attribute{Key: $2.Value.(string), Value: $4.Value.(string)}, nil)
} | AT IDENT {
	$$.Value = newElementTest("*", &Attribute{Key: $2.Value.(string)}, nil)
} | TEXT EQ STRING {
	$$.Value = newTextTest($3.Value.(string))
} | TEXT {
	$$.Value = newTextTest("")
};

OrExpr : AndExpr {}
  | OrExpr OR AndExpr {
	$$.Value = &predExpr{kind: "or", left: toExpr($1.Value), right: toExpr($3.Value)}
};
AndExpr : Factor {}
  | AndExpr AND Factor {
	$$.Value = &predExpr{kind: "and", left: toExpr($1.Value), right: toExpr($3.Value)}
};
// TODO(vvihorev): beware, Factor -> Path, and Factor -> Step semantic actions are getting dropped
Factor : Path {}
  | NOT Path {
	$$.Value = &predExpr{kind: "not", left: toExpr($2.Value)}
} | Step {}
  | NOT Step {
	$$.Value = &predExpr{kind: "not", left: toExpr($2.Value)}
} | LPAR OrExpr RPAR {
	$$.Value = $2.Value
} | NOT LPAR OrExpr RPAR {
	$$.Value = &predExpr{kind: "not", left: toExpr($3.Value)}
};

%%


// A step of a path, remembered so that the path can be rebuilt with the
// element carrying a predicate as its first step.
type pathStep struct {
	axis axis
	test udpeTest
}

type peSemValue struct {
	pe_id   int
	builder builder
	steps   []pathStep // every step of the path, in order
	mixed   bool       // the path mixes forward and reverse axes
}

// predExpr is a boolean expression over the operands of a predicate. Atoms can
// only be built once the element carrying the predicate is known, because the
// context of an atom is the node matched by the first step of its path.
type predExpr struct {
	kind        string // "and", "or", "not" or "leaf"
	left, right *predExpr
	leaf        any // *elementTest or *peSemValue
}

// toExpr wraps an operand of and/or/not. The generator removes the renaming
// rules Factor -> Path and Factor -> Step together with their actions, so an
// operand can still be a plain test or a path here.
func toExpr(v any) *predExpr {
	switch x := v.(type) {
	case *predExpr:
		return x
	case *elementTest, *peSemValue:
		return &predExpr{kind: "leaf", leaf: x}
	}
	panic(fmt.Sprintf("unexpected operand in a predicate: %T", v))
}

// buildPredicate turns an expression into a predicate of the element test
// 'carrier'.
func buildPredicate(e *predExpr, carrier *elementTest) *predicate {
	switch e.kind {
	case "and":
		return combine(and(), buildPredicate(e.left, carrier), buildPredicate(e.right, carrier))
	case "or":
		return combine(or(), buildPredicate(e.left, carrier), buildPredicate(e.right, carrier))
	case "not":
		return notNode(buildPredicate(e.left, carrier))
	}
	return newAtom(leafToID(e.leaf, carrier))
}

// leafToID registers the path of an operand and returns its id.
func leafToID(leaf any, carrier *elementTest) int {
	switch x := leaf.(type) {
	case *elementTest:
		if x.attr != nil {
			// an attribute test: the element it matches is the context itself
			return appendStep(nil, x, child).end()
		}
		return anchored(carrier, []pathStep{{child, x}})
	case *peSemValue:
		forward := len(x.steps) > 0 && (x.steps[0].axis == child || x.steps[0].axis == descendantOrSelf)
		if x.mixed || !forward {
			// paths with reverse axes are not anchored
			return x.end()
		}
		return anchored(carrier, x.steps)
	}
	panic(fmt.Sprintf("unexpected operand in a predicate: %T", leaf))
}

// anchored registers a path that starts with the carrier element.
func anchored(carrier *elementTest, steps []pathStep) int {
	pe := appendStep(nil, newElementTest(carrier.name, nil, nil), child)
	for _, s := range steps {
		pe = appendStep(pe, s.test, s.axis)
	}
	return pe.end()
}

func notNode(pred *predicate) *predicate {
	node := predNode{op: not()}
	pred.root.parent = &node
	node.left = pred.root
	pred.root = &node
	return pred
}

func newAtom(pe_id int) *predicate {
	node := predNode{op: atom()}
	return &predicate{root: &node, undoneAtoms: map[int]*predNode{pe_id: &node}}
}

func combine(op operator, dst, src *predicate) *predicate {
	node := predNode{op: op, left: dst.root, right: src.root}
	dst.root.parent = &node
	src.root.parent = &node
	for k, v := range src.undoneAtoms {
		dst.undoneAtoms[k] = v
	}
	dst.root = &node
	return dst
}

func (pe peSemValue) end() int {
	switch pe.builder.(type) {
	case *fpeBuilder:
		pe_id, _ := udpeGlobalTable.addFpe(pe.builder.(*fpeBuilder).end())
		return pe_id
	case *rpeBuilder:
		pe_id, _ := udpeGlobalTable.addRpe(pe.builder.(*rpeBuilder).end())
		return pe_id
	default:
		panic("something worong")
	}
}

func appendStep(pe *peSemValue, step udpeTest, axis axis) *peSemValue {
	if pe == nil {
		pe = &peSemValue{}
		if axis == child || axis == descendantOrSelf {
			pe.builder = &fpeBuilder{}
		} else {
			pe.builder = &rpeBuilder{}
		}
		pe.builder.addAxis(axis)
		pe.builder.addUdpeTest(step)
		pe.steps = []pathStep{{axis, step}}
		return pe
	}

	switch pe.builder.(type) {
	case *fpeBuilder:
		if axis == parent || axis == ancestorOrSelf {
			pe.pe_id, _ = udpeGlobalTable.addFpe(pe.builder.(*fpeBuilder).end())
			pe.builder = &rpeBuilder{}
			pe.mixed = true
		} else {
			pe.builder = pe.builder.(*fpeBuilder)
		}
	case *rpeBuilder:
		if axis == child || axis == descendantOrSelf {
			pe.pe_id, _ = udpeGlobalTable.addRpe(pe.builder.(*rpeBuilder).end())
			pe.builder = &fpeBuilder{}
			pe.mixed = true
		} else {
			pe.builder = pe.builder.(*rpeBuilder)
		}
	default:
		panic("something worong")
	}
	pe.builder.addAxis(axis)
	pe.builder.addUdpeTest(step)
	pe.steps = append(pe.steps, pathStep{axis, step})
	return pe
}

// ParserPreallocMem initializes all the memory pools required by the semantic function of the parser.
func ParserPreallocMem(inputSize int, numThreads int) {
}