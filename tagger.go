package pivotal_tracker_story_tagger
import (
	"os"
	"fmt"
	"net/http"

	"github.com/simplereach/timeutils"
	"io/ioutil"
	"encoding/json"
	"sort"
)

type Message struct {
	Id string `json:"id"`
	Name string `json:"name"`
	CreatedAt timeutils.Time `json:"created_at"`
}

type Messages []Message

func (slice Messages) Len() int {
	return len(slice)
}

func (slice Messages) Less(i, j int) bool {
	return slice[i].CreatedAt < slice[j].CreatedAt;
}

func (slice Messages) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func getPivotalTrackerEnvVariables() (projectId, apiToken, storyPrefix string) {
	projectId = os.Getenv("PIVOTAL_TRACKER_PROJECT_ID")
	apiToken = os.Getenv("PIVOTAL_TRACKER_API_TOKEN")
	storyPrefix = os.Getenv("PIVOTAL_TRACKER_STORY_PREFIX")
	return
}

func main() {
	const (
		PIVOTAL_TRACKER_API = "https://www.pivotaltracker.com/services/v5/projects/"
		PAGES = 2
		RESULTS_PER_PAGE = 500
	)

	projectId, apiToken, storyPrefix := getPivotalTrackerEnvVariables()

	var stories Messages

	if projectId == "" || apiToken == "" || storyPrefix == "" {
		fmt.Println("[ERROR]: Please set PIVOTAL_TRACKER_PROJECT_ID, PIVOTAL_TRACKER_API_TOKEN and " +
			"PIVOTAL_TRACKER_STORY_PREFIX")
		os.Exit(1)
	}

	client := &http.Client{}
	baseUrl := PIVOTAL_TRACKER_API + projectId

	for i := 0; i < PAGES; i++ {
		url := baseUrl + "stories?limit=" + string(RESULTS_PER_PAGE) + "&offset=" + string(RESULTS_PER_PAGE * i)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("[ERROR]: %s", err)
			os.Exit(1)
		}
		req.Header.Set("X-TrackerToken", apiToken)

		res, err := client.Do(req)
		if err != nil {
			fmt.Println("[ERROR]: %s", err)
			os.Exit(1)
		}

		contents, err := ioutil.ReadAll(res)
		if err != nil {
			fmt.Println("[ERROR]: %s", err)
			os.Exit(1)
		}

		var parsedJSON Message
		err = json.Unmarshal(contents, &parsedJSON)
		if err != nil {
			fmt.Println("[ERROR]: %s", err)
			os.Exit(1)
		}

		stories[i] = parsedJSON
	}

//	sorted_stories = stories.sort_by {|s| [s["id"], DateTime.iso8601(s["created_at"])]}
//	sort.Sort(stories)
//
//	var newlyTaggedStoryCount, maxSeenStoryId int
//
//	for story := range stories {
//		pivotalId := story.Id
//		storyName := story.Name
//
//		stringyStoryId := story
//	}
//
//	sorted_stories.each do |s|
//	pivotal_id = s["id"]
//	story_name = s["name"]
//	stringy_story_id = story_name[/\A#{Regexp.quote(STORY_PREFIX)}(\d+)/, 1]
//	if stringy_story_id
//	story_id = Integer(stringy_story_id)
//	max_seen_story_id = max_seen_story_id < story_id ? story_id : max_seen_story_id
//	log.debug "Skipping story with existing ID: #{story_name}"
//	next
//	end
//
//	newly_tagged_story_count += 1
//	max_seen_story_id += 1
//
//	new_story_name = "#{STORY_PREFIX}#{max_seen_story_id} - #{story_name}"
//	log.info "Adding new ID to story: #{new_story_name}"
//
//	# PUT the new name back into Pivotal.
//	project_api["stories/#{pivotal_id}"].put(({name: new_story_name}.to_json), {content_type: :json})
//	end
//
//	log.info "Tagged #{newly_tagged_story_count} new stories out of #{sorted_stories.length} total."
	fmt.Println("Tagged %d new stories out of %d", newlyTaggedStoryCount, len(stories))
}
