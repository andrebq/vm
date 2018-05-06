![master branch status](https://travis-ci.org/andrebq/vm.svg?branch=master)

# What is this?

A simple scripting language to allow Go applications to run untrusted code.

Speed is not the main concern, the language is desiged to be easly interoperable with the underlying platform, so if some block of code should be executed "fast" it should be implemented in the host language and exposed to the scripting language.

# Why?

So far, Lua has been the safest language to use for scripting, since it is possible to instrument the language to protect all important parts of the system. The major attach surface for a Lua interpreter is creating an infinite loop (which is mitigated by limiting the number of instructions a VM can execute without interrupts).

But the author has a taste for scheme-like languages and having being able to represent code and data with the same structures as binary and text streams has some extra value for the author (myself).

# Should I use it?

Only if you want to play and learn, right now most thins are work-in-progress.