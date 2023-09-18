package mockutil

type funcMatcher[P any] struct {
	desc    string
	check   func(P) error
	message string
}

// Func tests a function parameter P received by a mock provided a check
// function.
//
//	csvc.EXPECT().UpdateTransfer(
//		mockutil.Context(),
//		transferID,
//		mockutil.Func(
//			"should update X",
//			func(updater persistence.TransferUpdater) error {
//				_, err := updater(&goacontents.Transfer{})
//				return err
//			},
//		),
//	)
func Func[P any](description string, check func(P) error) *funcMatcher[P] {
	return &funcMatcher[P]{
		desc:  description,
		check: check,
	}
}

func (m *funcMatcher[P]) Matches(got interface{}) bool {
	err := m.check(got.(P))
	if err != nil {
		m.message = err.Error()
	}

	return err == nil
}

func (m funcMatcher[P]) String() string {
	return m.desc
}

func (m funcMatcher[P]) Got(got interface{}) string {
	return m.message
}
