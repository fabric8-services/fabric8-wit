/*
Package actions system is a key component for process automation in WIT. It provides a
way of executing user-configurable, dynamic process steps depending on user
settings, schema settings and events in the WIT.

The idea here is to provide a simple, yet powerful "signal-slot" system that
can connect any "event" in the system to any "action" with a clear decoupling of
events and actions with the goal of making the associations later dynamic and
configurable by the user ("user connects this event to this action"). Think
of a "IFTTT for WIT".

Actions are generic and atomic execution steps that do exactly one task and are
configurable. The actions system around the actions provide a key-based
execution of the actions.

Some examples for an application of this system would be:
  * closing all childs of a parent that is being closed (the user connects the
    "close" attribute change event of a WI to an action that closes all
    WIs of a matching query).
  * sending out notifications for mentions on markdown (the system executes
	  an action "send notification" for every mention found in markdown values).
  * moving all WIs from one iteration to the next in the time sequence when
    the original iteration is closed.

For all these automations, the actions system provides a re-usable, flexible and
later user configurable way of doing that without creating lots of custom code
and/or custom process implementations that are hardcoded in the WIT.

The current PR provides the basic actions infrastructure and an implementation of
an example action rules for testing.

This package provides two methods ExecuteActionsByOldNew() and ExecuteActionsByChangeset()
that can be called by a client (for example the controller on a request) with an entity
that is the context of the action run and a configuration for the context. The
configuration consists of a list of rule keys (identifying the rules that apply) and
respective configuration for the rules. The actions system will run the rules
sequentially and return the new context entity and a set of changes done while running
the rules. Note that executing actions may have sideffects on data beyond the context.
*/
package actions
