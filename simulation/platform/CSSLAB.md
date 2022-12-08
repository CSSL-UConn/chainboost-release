Navigation: [DEDIS](https://github.com/dedis/doc/tree/master/README.md) ::
[Onet](../../README.md) ::
[Simulation](../README.md) ::
Csslab

# Csslab Simulation

The first implementation of a simulation (even before localhost) was a csslab
simulation that allowed to run CoSi on https://csslab.net. Csslab is a
_state-of-the-art scientific computing facility for cyber-security researchers
engaged in research, development, discovery, experimentation, and testing of
innovative cyber-security technology_. It allows to reserve computers and define
the network between those. Using onet, it is very simple to pass from a localhost
simulation to a simulation using Csslab.

Before a successful Csslab simulation, you need to

1. be signed up at Csslab. If you're working with DEDIS, ask your
responsible for the _Project Name_ and the _Group Name_.
2. create a simulation defined by an NS-file. You can find a simple
NS-file here: [cothority.ns](./csslab_users/cothority.ns) - you'll need to
 adjust the # of servers, the type of servers, and the bandwidth- and delay
  restrictions.
3. swap the simulation in

For point 3. it is important of course that Csslab has enough machines
available at the moment of your experiment. If machines are missing, you might
change the NS-file to reference machines that are available. Attention: different
machines have different capabilities and can run more or less nodes, depending
on the power of the machine.

Supposing you have the points 1., 2., 3., solved, you can go on to the next step.

## Preparing automatic login

For easier simulation, you should prepare your ssh client to allow automatic
login to the remote server. If you login to the https://csslab.net website,
go to _My CssLab_ (on top), then _Profile_ (one of the tabs in the middle),
and finally chose _Edit SSH keys_ from the left. Now you can add your public
ssh-key to the list of allowed keys. Verify everything is running by checking
that you're not asked for your password when doing

```bash
ssh username@users.csslab.net
```

## Running a Csslab Simulation

If you have a successful [localhost](Localhost.md) simulation, it is very easy
to run it on Csslab. Make sure the experiment is swapped in, then you can
start it with:

```go
go build && ./simul -platform csslab simul.toml
```

Of course you'll need to adjust the `simul` and `simul.toml` to fit your
experiment. When running for the first time, the simulation will ask you for
your user-name, project and experiment. For the hostname of csslab and the
monitor address, you can accept the default values. It will store all this in the
`cssl.toml` file, where you can also change it. Or you delete the file, if you
want to enter the information again.

## Monitoring Port

During the experiment, your computer will play the role as the organizer and
start all the experiments on the remote csslab network. This means, that
your computer needs to be online during the whole experiment.

To communicate with Csslab, the simulation starts a ssh-tunnel to csslab.net
where all information from the simulation are received. As there might
be more than one person using csslab, and all users need to go through
users.csslab.net, you might want to change the port-number you're using. It's
as easy as:

```go
./simul -platform csslab -mport 10002 simult.toml
```

This will use port 10002 to communicate with csslab. Be sure to increment by
2, so only use even numbers.

## Oversubscription

If you have more nodes than available servers, the simulation will put multiple
cothority-nodes on the same server. When creating the `Tree` in
`Simulation.Setup`, it will make sure that any parent-children connection will
go through the network, so that every communication between a parent and a
child will go through the bandwidth and timing restrictions.

Unfortunately this means that not all nodes will have these restrictions, and
in the case of a broadcast with oversubscription, some nodes might communicate
directly with the other nodes. If you need a fully restricted network, you
need to use [Mininet](MININET.md)
