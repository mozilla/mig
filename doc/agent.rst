Agent
=====

The agent accepts different classes of inputs on stdin, as one-line JSON objects. The most common one is the
``parameters`` class, but it could also receive a ``stop`` input that
indicates that the module should stop its execution immediately. The format of
module input messages is defined by ``modules.Message``.

.. code:: go

	// Message defines the input messages received by modules.
	type Message struct {
		Class      string      // represent the type of message being passed to the module
		Parameters interface{} // for `parameters` class, this interface contains the module parameters
	}

	const (
		MsgClassParameters string = "parameters"
		MsgClassStop       string = "stop"
	)

When the agent receives a command to pass to a module for execution, it
extracts the operation parameters from ``Command.Action.Operations[N].Parameters``
and copies them into ``Message.Parameters``. It then sets ``Message.Class`` to
``modules.MsgClassParameters``, marshals the struct into JSON, and pass the
resulting ``[]byte`` to the module as an IO stream.


