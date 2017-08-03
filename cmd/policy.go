package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"go.uber.org/zap"

	"github.com/golang/protobuf/jsonpb"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitlab.com/SporeDB/sporedb/db"
)

var policyPath *string

// policyCmd represents the policy command
var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage SporeDB policies",
}

var policyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new policy",
	Run: func(cmd *cobra.Command, args []string) {
		keyRing := getKeyRing()
		p := &db.Policy{}
		p.Uuid = read("Name of the policy", uuid.NewV4().String())
		p.Comment = read("Comment", "")
		p.Specs = []*db.OSpec{
			&db.OSpec{Key: &db.OSpec_Regex{Regex: ".*"}},
		}

		if readBool("Shall this node be considered as an endorser?", true) {
			data, _, _ := keyRing.GetPublic("")
			p.Endorsers = append(p.Endorsers, &db.Endorser{Public: data})
		}

		var i int
		for {
			i++
			name := read("Endorser #"+strconv.Itoa(i)+" (blank to skip)", "")
			if name == "" {
				break
			}

			data, _, err := keyRing.GetPublic(name)
			if err != nil {
				i--
				fmt.Println(err)
				continue
			}

			p.Endorsers = append(p.Endorsers, &db.Endorser{Public: data, Comment: name})
		}

		f := readInt("Maximum number of byzantine (faulty) endorsers", 1)

		quorum := 1 + (len(p.Endorsers)+f)/2
		userQuorum := readInt("Quorum", quorum)
		if userQuorum < quorum {
			check(fmt.Errorf("insufficient quorum, got %d but at least %d endorsers are required", len(p.Endorsers), quorum))
		}
		if userQuorum > len(p.Endorsers) {
			check(fmt.Errorf("insufficient number of endorsers"))
		}

		p.Quorum = uint64(userQuorum)
		m := &jsonpb.Marshaler{EmitDefaults: true, Indent: "  ", OrigName: true}
		s, err := m.MarshalToString(p)
		check(err)

		check(ioutil.WriteFile(path.Join(*policyPath, p.Uuid+".json"), []byte(s), 0600))
	},
}

func init() {
	policyPath = policyCreateCmd.Flags().StringP("path", "p", ".", "policies location")
	policyCmd.AddCommand(policyCreateCmd)
	RootCmd.AddCommand(policyCmd)
}

func loadPolicies(database *db.DB) {
	for _, p := range viper.GetStringSlice("db.policies") {
		f, err := os.Open(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to load", p, "(cannot open file)")
			continue
		}
		defer func() { _ = f.Close() }()

		u := &jsonpb.Unmarshaler{}
		policy := &db.Policy{}
		err = u.Unmarshal(f, policy)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to parse", p, ":", err)
			continue
		}

		err = database.AddPolicy(policy)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to enable", p, ":", err)
			continue
		}

		zap.L().Info("Loaded policy",
			zap.String("uuid", policy.Uuid),
		)
	}
}
