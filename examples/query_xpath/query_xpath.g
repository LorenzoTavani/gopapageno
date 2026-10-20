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
		// handle renaming chain OrExpr -> AndExpr -> Factor -> Step
		switch $3.Value.(type) {
		case *elementTest:
			pe := appendStep(nil, $3.Value.(udpeTest), child)
			$3.Value = newAtom(pe.end())
		case *peSemValue:
			$3.Value = newAtom($3.Value.(*peSemValue).end())
		}

		$1.Value.(*elementTest).pred = $3.Value.(*predicate)
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
		// handle renaming chain OrExpr -> AndExpr -> Factor -> Step
		switch $3.Value.(type) {
		case *elementTest:
			pe := appendStep(nil, $3.Value.(udpeTest), child)
			$3.Value = newAtom(pe.end())
		case *peSemValue:
			$3.Value = newAtom($3.Value.(*peSemValue).end())
		}

		$1.Value.(*elementTest).pred = $3.Value.(*predicate)
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
  | AndExpr OR AndExpr {
	$$.Value = combine(or(), toPredicate($1.Value), toPredicate($3.Value))
};

AndExpr : Factor {}
  | Factor AND Factor {
	$$.Value = combine(and(), toPredicate($1.Value), toPredicate($3.Value))
};

// TODO(vvihorev): beware, Factor -> Path, and Factor -> Step semantic actions are getting dropped
Factor : Path {
	$$.Value = newAtom($1.Value.(*peSemValue).end())
} | NOT Path {
	$$.Value = notNode(newAtom($2.Value.(*peSemValue).end()))
} | Step {}
  | NOT Step {
	step := appendStep(nil, $2.Value.(udpeTest), child)
	$$.Value = notNode(newAtom(step.end()))
} | LPAR OrExpr RPAR {
	$$.Value = $2.Value
} | NOT LPAR OrExpr RPAR {
	$$.Value = notNode(toPredicate($3.Value))
};

%%

type peSemValue struct {
	pe_id   int
	builder builder
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

// toPredicate turns an operand of and/or/not into a predicate. The generator
// removes the renaming rules Factor -> Path and Factor -> Step together with
// their actions, so an operand can still be a plain test or a path here.
func toPredicate(v any) *predicate {
	switch x := v.(type) {
	case *predicate:
		return x
	case *elementTest:
		return newAtom(appendStep(nil, x, child).end())
	case *peSemValue:
		return newAtom(x.end())
	}
	panic(fmt.Sprintf("unexpected operand in a predicate: %T", v))
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
		return pe
	}

	switch pe.builder.(type) {
	case *fpeBuilder:
		if axis == parent || axis == ancestorOrSelf {
			pe.pe_id, _ = udpeGlobalTable.addFpe(pe.builder.(*fpeBuilder).end())
			pe.builder = &rpeBuilder{}
		} else {
			pe.builder = pe.builder.(*fpeBuilder)
		}
	case *rpeBuilder:
		if axis == child || axis == descendantOrSelf {
			pe.pe_id, _ = udpeGlobalTable.addRpe(pe.builder.(*rpeBuilder).end())
			pe.builder = &fpeBuilder{}
		} else {
			pe.builder = pe.builder.(*rpeBuilder)
		}
	default:
		panic("something worong")
	}

	pe.builder.addAxis(axis)
	pe.builder.addUdpeTest(step)
	return pe
}

// var parserElementsPools []*gopapageno.Pool[xpath.Element]

// ParserPreallocMem initializes all the memory pools required by the semantic function of the parser.
func ParserPreallocMem(inputSize int, numThreads int) {
    // poolSizePerThread := 10000

    // parserElementsPools = make([]*gopapageno.Pool[xpath.Element], numThreads)
    // for i := 0; i < numThreads; i++ {
    //     parserElementsPools[i] = gopapageno.NewPool[xpath.Element](poolSizePerThread)
    // }
}
