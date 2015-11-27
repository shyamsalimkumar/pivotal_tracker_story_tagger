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
    "io"
    "bytes"
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
    Name string `json:"name,omitempty"`
}

func getPivotalTrackerEnvVariables() (projectId, apiToken, storyPrefix string) {
    projectId = os.Getenv("PIVOTAL_TRACKER_PROJECT_ID")
    apiToken = os.Getenv("PIVOTAL_TRACKER_API_TOKEN")
    storyPrefix = os.Getenv("PIVOTAL_TRACKER_STORY_PREFIX")
    return
}

func makeRequest(client *http.Client, url string, method string, apiToken string, data io.Reader) (contents []byte, err error) {
    req, err := http.NewRequest(method, url, data)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-TrackerToken", apiToken)

    res, err := client.Do(req)
    defer res.Body.Close()
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }

    contents, err = ioutil.ReadAll(res.Body)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }
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

        contents, err := makeRequest(client, listUrl, "GET", apiToken, nil)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }

        var storiesFromJson Messages
        err = json.Unmarshal(contents, &storiesFromJson)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }
        stories = append(stories, storiesFromJson...)
    }

    sort.Sort(stories)
    regex := "" + storyPrefix + "(\\d+)"
    re, err := regexp.Compile(regex)
    if err != nil {
        fmt.Println("[ERROR]:", err)
        os.Exit(1)
    }

    var newlyTaggedStoryCount, maxSeenStoryId int

    for _, story := range stories {
        pivotalId := story.Id
        storyName := story.Name

        matches := re.FindStringSubmatch(storyName)
        if len(matches) > 0 {
            stringyStoryId := matches[1]
            storyId, err := strconv.Atoi(stringyStoryId)
            if err != nil {
                fmt.Println("[ERROR]:", err)
                os.Exit(1)
            }
            if maxSeenStoryId < storyId {
                maxSeenStoryId = storyId
            }
            fmt.Println("Skipping story with existing ID:", storyName)
            continue
        }

        newlyTaggedStoryCount += 1
        maxSeenStoryId += 1

        newStoryName := storyPrefix + strconv.Itoa(maxSeenStoryId) +" - " + storyName
        fmt.Println("Adding new ID to story:", newStoryName)

        updateUrl := baseUrl + "stories/" + strconv.Itoa(pivotalId)
        putData := &OutboundMessage{Name: newStoryName}
        b, err := json.Marshal(putData)
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }

        contents, err := makeRequest(client, updateUrl, "PUT", apiToken, bytes.NewBuffer(b))
        if err != nil {
            fmt.Println("[ERROR]:", err)
            os.Exit(1)
        }
        fmt.Println(string(contents))
    }

    fmt.Printf("Tagged %d new stories out of %d total", newlyTaggedStoryCount, len(stories))
}
