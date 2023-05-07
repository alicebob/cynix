with rec { pkgs = import <nixpkgs> { }; };

pkgs.mkShell {
  buildInputs = [
    pkgs.go_1_20
  ];

  shellHook = ''
    echo hello
  '';
}

