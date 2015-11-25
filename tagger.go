package main
import (
	"os"
	"fmt"
	"net/http"

	"io/ioutil"
	"encoding/json"
	"sort"
	"time"
    "regexp"
    "strconv"
)

type Message struct {
	Id int `json:"id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Messages []Message

func (slice Messages) Len() int {
	return len(slice)
}

func (slice Messages) Less(i, j int) bool {
	return slice[i].CreatedAt.Before(slice[j].CreatedAt);
}

func (slice Messages) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type OutboundMessage struct {
    name string
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
	baseUrl := PIVOTAL_TRACKER_API + projectId + "/"

	for i := 0; i < PAGES; i++ {
		listUrl := baseUrl + "stories?limit=" + strconv.Itoa(RESULTS_PER_PAGE) + "&offset=" + strconv.Itoa(RESULTS_PER_PAGE * i)
		listReq, err := http.NewRequest("GET", listUrl, nil)
		if err != nil {
			fmt.Println("[ERROR]:", err)
			os.Exit(1)
		}
		listReq.Header.Set("X-TrackerToken", apiToken)

		res, err := client.Do(listReq)
		defer res.Body.Close()
		if err != nil {
			fmt.Println("[ERROR]:", err)
			os.Exit(1)
		}

		contents, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("[ERROR]:", err)
			os.Exit(1)
		}

		err = json.Unmarshal(contents, &stories)
		if err != nil {
			fmt.Println("[ERROR]:", err)
			os.Exit(1)
		}
	}

	sort.Sort(stories)
    regex := "/" + storyPrefix + "(\\d+)/"
    re, err := regexp.Compile(regex)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }

	var newlyTaggedStoryCount, maxSeenStoryId int

	fmt.Println("Number of stories found:", len(stories))
    for _, story := range stories {
//		pivotalId := story.Id
		storyName := story.Name

		matches := re.FindStringSubmatch(storyName)
        fmt.Printf("Matches found %d", len(matches))
        if len(matches) > 0 {
            stringyStoryId := matches[0]
            storyId, err := strconv.Atoi(stringyStoryId)
            if err != nil {
                fmt.Println("[ERROR]:", err)
                os.Exit(1)
            }
            if maxSeenStoryId < storyId {
                maxSeenStoryId = storyId
            }
            fmt.Println("Skipping story with existing ID: #%s", storyName)
        }

        newlyTaggedStoryCount += 1
        maxSeenStoryId += 1

        newStoryName := "#" + storyPrefix + "#" + string(maxSeenStoryId) +" - #" + string(maxSeenStoryId)
        fmt.Println("Adding new ID to story:", newStoryName)

//        updateUrl := PIVOTAL_TRACKER_API + "stories/#" + string(pivotalId)
//        addReq, err := http.NewRequest("PUT", updateUrl, nil)
//        if err != nil {
//            fmt.Println("[ERROR]:", err)
//            os.Exit(1)
//        }
//        addReq.Header.Set("X-TrackerToken", apiToken)
//
//        res, err := client.Do(addReq)
//        defer res.Body.Close()
//        if err != nil {
//            fmt.Println("[ERROR]:", err)
//            os.Exit(1)
//        }
	}

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
	fmt.Printf("Tagged %d new stories out of %d total", newlyTaggedStoryCount, len(stories))
}
